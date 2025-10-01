package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/repositories"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/services"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases/input"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases/output"
)

type RoomLocker interface {
	WithLock(ctx context.Context, roomID uuid.UUID, fn func(ctx context.Context) error) error
}

var errQueueNotEmpty = errors.New("待機列が存在するため即時入場できません")

type EnterWaitingRoomUseCase struct {
	waitingRoomRepository repositories.WaitingRoomRepository
	userRepository        repositories.UserRepository
	sessionRepository     repositories.SessionRepository
	ticketRepository      repositories.TicketRepository
	ticketLifecycle       services.TicketLifecycle
	sessionPolicy         services.SessionPolicy
	roomLocker            RoomLocker
}

func NewEnterWaitingRoomUseCase(
	waitingRoomRepository repositories.WaitingRoomRepository,
	userRepository repositories.UserRepository,
	sessionRepository repositories.SessionRepository,
	ticketRepository repositories.TicketRepository,
	ticketLifecycle services.TicketLifecycle,
	sessionPolicy services.SessionPolicy,
	roomLocker RoomLocker,
) *EnterWaitingRoomUseCase {
	return &EnterWaitingRoomUseCase{
		waitingRoomRepository: waitingRoomRepository,
		userRepository:        userRepository,
		sessionRepository:     sessionRepository,
		ticketRepository:      ticketRepository,
		ticketLifecycle:       ticketLifecycle,
		sessionPolicy:         sessionPolicy,
		roomLocker:            roomLocker,
	}
}

// 待機室への入場処理
func (u *EnterWaitingRoomUseCase) Execute(ctx context.Context, in input.EnterWaitingRoomInput) (output.EnterWaitingRoomOutput, error) {
	var result output.EnterWaitingRoomOutput
	// 依存チェック
	if u == nil {
		return result, errors.New("待機室入場ユースケースが未設定です")
	}
	if u.ticketLifecycle == nil {
		return result, errors.New("チケットライフサイクルが未設定です")
	}
	if u.sessionPolicy == nil {
		return result, errors.New("セッションポリシーが未設定です")
	}

	// 状態遷移の時刻を揃えるため処理全体で共通の時刻を私用
	now := in.EffectiveTime()

	// 待機室のキャパや現在のキュー状況を取得
	room, err := u.waitingRoomRepository.GetByID(ctx, in.WaitingRoomID)
	if err != nil {
		return result, err
	}

	// ユーザー状態を取得、見つからなければ待機中ユーザーとして新規登録
	user, err := u.userRepository.GetByID(ctx, in.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			created := entities.NewQueuedUser(in.UserID, in.UserName, now)
			user = &created
		} else {
			return result, err
		}
	} else if in.UserName != nil {
		user.Name = in.UserName
	}

	// 有効なセッションがあるなら入場済みなのでそのまま返す
	if session, err := u.sessionRepository.GetActiveByUserID(ctx, in.UserID); err == nil {
		result.Outcome = output.EnterWaitingRoomOutcomeEnterTarget
		result.SessionID = &session.ID
		return result, nil
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return result, err
	}

	// セッションが無ければチケットを調べ、再入場できるか継続待機かを決める
	var ticket *entities.Ticket
	if t, err := u.ticketRepository.GetActiveByUserID(ctx, in.UserID); err == nil {
		ticket = t
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return result, err
	}

	if ticket != nil {
		// 期限切れのチケットは失効扱いにして再度待機列へ戻す
		if ticket.IsExpired(now) {
			u.ticketLifecycle.MarkExpired(ticket)
			if err := u.ticketRepository.Save(ctx, ticket); err != nil {
				return result, err
			}
			ensureQueued(room, in.UserID)
			if err := u.waitingRoomRepository.Save(ctx, room); err != nil {
				return result, err
			}
			user.ResetToWaiting(now)
			if err := u.userRepository.Save(ctx, user); err != nil {
				return result, err
			}
			result.Outcome = output.EnterWaitingRoomOutcomeRedirectWaitingRoom
			result.TicketID = &ticket.ID
			return result, nil
		}

		if ticket.Status != entities.TicketStatusAdmitted {
			// まだ入場前のチケットなら待機中に揃え、ユーザーもキューへ入れる
			if ticket.Status != entities.TicketStatusWaiting {
				u.ticketLifecycle.MarkWaiting(ticket)
				if err := u.ticketRepository.Save(ctx, ticket); err != nil {
					return result, err
				}
			}
			ensureQueued(room, in.UserID)
			if err := u.waitingRoomRepository.Save(ctx, room); err != nil {
				return result, err
			}
			if user.Status != entities.UserStatusWaiting {
				user.ResetToWaiting(now)
				if err := u.userRepository.Save(ctx, user); err != nil {
					return result, err
				}
			}
			result.Outcome = output.EnterWaitingRoomOutcomeRedirectWaitingRoom
			result.TicketID = &ticket.ID
			return result, nil
		}

		// 順番が来たチケットなのでロック下で利用・セッション発行・キュー整理を行う
		var session entities.Session
		err := u.withRoomLock(ctx, room.ID, func(lockCtx context.Context) error {
			if err := u.ticketLifecycle.ValidateAndUse(ticket, now); err != nil {
				return err
			}
			session = u.sessionPolicy.Create(in.UserID, now)
			if err := u.sessionRepository.Save(lockCtx, &session); err != nil {
				return err
			}
			room.Remove(in.UserID)
			if err := u.waitingRoomRepository.Save(lockCtx, room); err != nil {
				return err
			}
			if err := u.ticketRepository.Save(lockCtx, ticket); err != nil {
				return err
			}
			user.MarkEntered(now)
			if err := u.userRepository.Save(lockCtx, user); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return result, err
		}
		result.Outcome = output.EnterWaitingRoomOutcomeEnterTarget
		result.SessionID = &session.ID
		result.TicketID = &ticket.ID
		result.NewlyIssuedSession = true
		return result, nil
	}

	// チケットを持たないユーザーは、待機者ゼロかつ空きがあるときだけ即入場
	if room.HasCapacity() && room.Len() == 0 {
		var session entities.Session
		err := u.withRoomLock(ctx, room.ID, func(lockCtx context.Context) error {
			if room.Len() > 0 {
				return errQueueNotEmpty
			}
			session = u.sessionPolicy.Create(in.UserID, now)
			if err := u.sessionRepository.Save(lockCtx, &session); err != nil {
				return err
			}
			room.Remove(in.UserID)
			if err := u.waitingRoomRepository.Save(lockCtx, room); err != nil {
				return err
			}
			user.MarkEntered(now)
			if err := u.userRepository.Save(lockCtx, user); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, errQueueNotEmpty) {
				return u.issueTicket(ctx, in, room, user, now)
			}
			return result, err
		}
		result.Outcome = output.EnterWaitingRoomOutcomeEnterTarget
		result.SessionID = &session.ID
		result.NewlyIssuedSession = true
		return result, nil
	}

	// それ以外はチケットを新発行して順番待ちさせる
	return u.issueTicket(ctx, in, room, user, now)
}

func (u *EnterWaitingRoomUseCase) issueTicket(
	ctx context.Context,
	in input.EnterWaitingRoomInput,
	room *entities.WaitingRoom,
	user *entities.User,
	now time.Time,
) (output.EnterWaitingRoomOutput, error) {
	var result output.EnterWaitingRoomOutput
	issued := u.ticketLifecycle.Issue(in.UserID, room.TTL, now)
	ticket := &issued
	err := u.withRoomLock(ctx, room.ID, func(lockCtx context.Context) error {
		if err := u.ticketRepository.Save(lockCtx, ticket); err != nil {
			return err
		}
		ensureQueued(room, in.UserID)
		if err := u.waitingRoomRepository.Save(lockCtx, room); err != nil {
			return err
		}
		user.ResetToWaiting(now)
		if err := u.userRepository.Save(lockCtx, user); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	result.Outcome = output.EnterWaitingRoomOutcomeRedirectWaitingRoom
	result.TicketID = &ticket.ID
	result.NewlyIssuedTicket = true
	return result, nil
}

func (u *EnterWaitingRoomUseCase) withRoomLock(ctx context.Context, roomID uuid.UUID, fn func(context.Context) error) error {
	if u.roomLocker == nil {
		return fn(ctx)
	}
	return u.roomLocker.WithLock(ctx, roomID, fn)
}

func ensureQueued(room *entities.WaitingRoom, userID uuid.UUID) {
	if room == nil {
		return
	}
	for _, id := range room.Queue {
		if id == userID {
			return
		}
	}
	room.Enqueue(userID)
}
