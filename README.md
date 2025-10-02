# simple-virtual-waiting-room
勉強用シンプル仮想待機室

## 超簡易入場フロー図
![IH-ScreenShot 2025-09-28 at 21 31 50](https://github.com/user-attachments/assets/66ff4b60-38bd-4130-9022-227416a7938b)

## 定員管理の仕組み
- `TARGET_MAX_ACTIVE_SESSIONS` でターゲットに入場できる同時アクティブセッション数を設定
- Valkeyの `session:active` セットを確認し、上限に達していれば新しいユーザーは待機列に飛ばす
- セッションには `SESSION_TTL_SECONDS` で指定したTTLを付与、`POST /waiting-room/session/heartbeat` でハートビートを送ると延長
- TTLが切れるとセッションを削除し、枠を開ける
- 既存セッションを持つユーザーが再アクセスした場合は即入場扱いとなり、ハートビートが届く限り枠は維持

## ディレクトリ構成
```
.
├─ cmd/waiting-room        # エントリポイント（main）
├─ internal
│  ├─ app                  # 依存注入・HTTP サーバー・ルーター
│  ├─ config               # 環境変数読み込み
│  ├─ middleware           # 共通ミドルウェア
│  └─ waiting-room
│     ├─ controllers       # HTTP ハンドラ（API/ブラウザ向け）
│     ├─ domain            # エンティティとドメインリポジトリ定義
│     ├─ infrastructure    # Valkey など外部リソース実装
│     ├─ presenters        # レスポンス生成
│     ├─ services          # ドメインサービス（セッション/チケット）
│     └─ usecases          # ビジネスロジック（待機室入場など）
├─ apps/sample-target-app  # デモ用ターゲットアプリ
└─ Dockerfile / docker-compose.yml / Makefile など
```
