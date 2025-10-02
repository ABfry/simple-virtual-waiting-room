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

var (
	errQueueNotEmpty   = errors.New("待機列が存在するため即時入場できません")
	errCapacityReached = errors.New("ターゲットサービスの定員に達しています")
)

type EnterWaitingRoomUseCase struct {
	waitingRoomID         uuid.UUID
	waitingRoomRepository repositories.WaitingRoomRepository
	userRepository        repositories.UserRepository
	sessionRepository     repositories.SessionRepository
	ticketRepository      repositories.TicketRepository
	ticketLifecycle       services.TicketLifecycle
	sessionPolicy         services.SessionPolicy
	roomLocker            RoomLocker
	maxActiveSessions     int
	sessionTTL            time.Duration
}

func NewEnterWaitingRoomUseCase(
	waitingRoomID uuid.UUID,
	waitingRoomRepository repositories.WaitingRoomRepository,
	userRepository repositories.UserRepository,
	sessionRepository repositories.SessionRepository,
	ticketRepository repositories.TicketRepository,
	ticketLifecycle services.TicketLifecycle,
	sessionPolicy services.SessionPolicy,
	roomLocker RoomLocker,
	maxActiveSessions int,
	sessionTTL time.Duration,
) *EnterWaitingRoomUseCase {
	return &EnterWaitingRoomUseCase{
		waitingRoomID:         waitingRoomID,
		waitingRoomRepository: waitingRoomRepository,
		userRepository:        userRepository,
		sessionRepository:     sessionRepository,
		ticketRepository:      ticketRepository,
		ticketLifecycle:       ticketLifecycle,
		sessionPolicy:         sessionPolicy,
		roomLocker:            roomLocker,
		maxActiveSessions:     maxActiveSessions,
		sessionTTL:            sessionTTL,
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
	room, err := u.waitingRoomRepository.GetByID(ctx, u.waitingRoomID)
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
		if err := u.refreshSession(ctx, session); err != nil {
			return result, err
		}
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

		// 枠が空いている場合は待機チケットを昇格させる
		if ticket.Status == entities.TicketStatusWaiting {
			front := room.Peek(1)
			if len(front) > 0 && front[0] == in.UserID {
				err := u.withRoomLock(ctx, room.ID, func(lockCtx context.Context) error {
					if err := u.ensureCapacity(lockCtx); err != nil {
						return err
					}
					u.ticketLifecycle.MarkAdmitted(ticket)
					return u.ticketRepository.Save(lockCtx, ticket)
				})
				if err != nil {
					if errors.Is(err, errCapacityReached) {
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
					return result, err
				}
			}
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

		// 順番が来たチケットなのでロック下で定員を確認しつつ利用・セッション発行・キュー整理を行う
		var session entities.Session
		err := u.withRoomLock(ctx, room.ID, func(lockCtx context.Context) error {
			if err := u.ensureCapacity(lockCtx); err != nil {
				if errors.Is(err, errCapacityReached) {
					u.ticketLifecycle.MarkWaiting(ticket)
					if err := u.ticketRepository.Save(lockCtx, ticket); err != nil {
						return err
					}
					ensureQueued(room, in.UserID)
					if err := u.waitingRoomRepository.Save(lockCtx, room); err != nil {
						return err
					}
					if user.Status != entities.UserStatusWaiting {
						user.ResetToWaiting(now)
						if err := u.userRepository.Save(lockCtx, user); err != nil {
							return err
						}
					}
				}
				return err
			}
			if err := u.ticketLifecycle.ValidateAndUse(ticket, now); err != nil {
				return err
			}
			session = u.sessionPolicy.Create(in.UserID, now)
			if err := u.sessionRepository.Save(lockCtx, &session); err != nil {
				return err
			}
			if err := u.refreshSession(lockCtx, &session); err != nil {
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
			if errors.Is(err, errCapacityReached) {
				result.Outcome = output.EnterWaitingRoomOutcomeRedirectWaitingRoom
				result.TicketID = &ticket.ID
				return result, nil
			}
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
			if err := u.ensureCapacity(lockCtx); err != nil {
				return err
			}
			session = u.sessionPolicy.Create(in.UserID, now)
			if err := u.sessionRepository.Save(lockCtx, &session); err != nil {
				return err
			}
			if err := u.refreshSession(lockCtx, &session); err != nil {
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
			if errors.Is(err, errQueueNotEmpty) || errors.Is(err, errCapacityReached) {
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

func (u *EnterWaitingRoomUseCase) ensureCapacity(ctx context.Context) error {
	if u.maxActiveSessions <= 0 {
		return nil
	}
	count, err := u.sessionRepository.CountActive(ctx)
	if err != nil {
		return err
	}
	if count >= int64(u.maxActiveSessions) {
		return errCapacityReached
	}
	return nil
}

func (u *EnterWaitingRoomUseCase) refreshSession(ctx context.Context, session *entities.Session) error {
	if session == nil || u.sessionTTL <= 0 {
		return nil
	}
	return u.sessionRepository.RefreshTTL(ctx, session, u.sessionTTL)
}

// KeepSessionAlive はアクティブセッションの TTL を延長し続ける。
func (u *EnterWaitingRoomUseCase) KeepSessionAlive(ctx context.Context, sessionID uuid.UUID) error {
	if u.sessionTTL <= 0 {
		return nil
	}
	session, err := u.sessionRepository.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if !session.IsActive() {
		return repositories.ErrNotFound
	}
	return u.refreshSession(ctx, session)
}
