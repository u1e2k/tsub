# Linux コンソール・uim-fep (Anthy) 環境における Preedit 重複描画問題の調査報告と解決設計

## 1. 概要 (Executive Summary)

Raspberry Pi 等の小型ディスプレイ（画面行数 12〜14 行程度のフレームバッファ / Linux 仮想コンソール）環境において、日本語入力フロントエンドプロセッサである `uim-fep` (Anthy-utf8) を介して `tsub` を実行した際、**日本語の未確定文字列（Preedit / かな漢字変換中の文字列）が入力ボックスではなく最下部のフッター行（`[入力] Enter: 投稿 ... 終了`）の上に直接上書き描画される**という現象が発生しました。

さらに未確定文字列が長くなると、端末の行末折り返しや改行処理によって画面全体が上スクロールし、ヘッダーや入力欄自体が画面外（上部）へ押し出されてしまう問題も併発していました。

本ドキュメントでは、`uim-fep` のソースコード解析によって判明した描画メカニズム、TUI フレームワーク（Bubble Tea / Lipgloss）との相互作用による根本原因、および完全な解消に至る設計と実装の詳細を後学のために記録します。

---

## 2. 発生していた現象

### 症状の視覚的イメージ
小型 LCD（例: 縦 13 行）上で日本語入力モードに切り替え、`この部分がかぶらないようにする方法はないのか` とタイピングした際の画面：

```text
┌────────────────────────────────────────────────────────┐
│ 20:21 テスト                                           │ ← 画面が上スクロールし
│ 20:01 おお、ちゃんと続きから読み込むのか。             │   ヘッダーや入力枠が
│ 19:58 あああああああああああああああああ               │   上へ押し出される
│ 19:57 え、すご。                                       │
│                                                        │
│ この部分がかぶらないようにする方法はないのか      終了 │ ← フッターの上に重なる！
│                                                        │
│ anthy-utf8[AnあR]                                      │ ← uim-fep ステータス行
└────────────────────────────────────────────────────────┘
```

- 下部にマージン行（余白）を 2〜4 行設けても、**なぜか常にフッターの文字（`終了` など）の上に未確定文字列が重なる**。
- 本来文字を入力すべきエディタ枠内（`💭 いまどうしてる？`）には一切変換中の文字が出ない。

---

## 3. 根本原因の技術的分析 (Root Cause Analysis)

### 3.1 `uim-fep` の Preedit 描画メカニズム
`uim-fep` のソースコード (`fep/draw.c`) を調査した結果、未確定文字列の描画開始位置は以下のように決定されていました。

```c
/* uim/fep/draw.c より抜粋 */
static void start_preedit(void)
{
  if (!g_start_preedit) {
    g_start_preedit = TRUE;
    if (g_opt.no_report_cursor) {
      return;
    }

    /* 端末へエスケープシーケンス (CSI 6 n) を送信し、現在のハードウェアカーソル位置を取得 */
    s_head = get_cursor_position();
    ...
```

1. ユーザーが日本語のキーを入力すると、`uim-fep` は ANSI Device Status Report (`\x1b[6n`) を端末に送信し、**「現在の端末ハードウェアカーソルがどこにあるか」** を問い合わせます。
2. 端末から返ってきた座標 `s_head (row, col)` を **Preedit の描画開始地点** として記憶します。
3. `s_head` から順に、未確定文字列（下線付き文字等）を端末へ直接書き出します。

### 3.2 TUI (Bubble Tea / Lipgloss) 側の動作
Bubble Tea などの多くの TUI フレームワークでは、`View()` メソッドが画面全体の複数行文字列（ANSI エスケープシーケンス付き文字列）を返し、それをターミナルへ一括出力します。

`tsub` の描画順序：
```
1. HeaderBar       (行 1)
2. EditorBox       (行 2〜5)
3. TimelineView    (行 6〜9)
4. FooterBar       (行 10)  ← [入力] Enter: 投稿 ... 終了
```

`View()` の出力文字列は `FooterBar` で終わるため、端末の**ハードウェアカーソルはフッター行の末尾に置かれたまま**待機状態になります。

### 3.3 なぜ被ってしまっていたのか？
- `uim-fep` は「アプリケーションの論理的な入力欄がどこにあるか」を自動的には知り得ません。
- 頼りにできる唯一の情報は**「端末のハードウェアカーソル位置」**だけです。
- `tsub` がハードウェアカーソルを移動させずに放置していたため、`uim-fep` にとっては**「カーソルがフッター行にあるのだから、ユーザーはフッター行に入力したいのだ」**と判断され、フッター行から未確定文字列を描画し始めました。
- さらに、小型画面で Preedit 文字列が横幅を超えたり行末に達すると、端末が自動改行・スクロールを発生させ、画面上部にあったヘッダーやエディタ枠を押し流してしまっていたのです。

---

## 4. 解決策の設計と実装 (Solution Architecture)

解決アプローチは**「モードに応じた正確なハードウェアカーソル座標制御」**です。

### 4.1 入力モード (`ModeInput`)
ユーザーが入力枠にフォーカスしているときは、エディタ枠内の現在入力行・列へハードウェアカーソルを移動させます。

#### 座標の正確な計算
端末の画面座標系（1-indexed: `(row, col) = (1, 1)` が左上）に合わせた計算式：

1. **行位置 (`targetY`)**:
   ```go
   // 1 (基準) + ヘッダー高 + スペーサー + タイトル高 + 枠線上ボーダー(1) + テキストエリア内行オフセット
   targetY := 1 + headerHeight + spacer1 + editorTitleHeight + 1 + m.textarea.LineInfo().RowOffset
   ```
2. **列位置 (`targetX`)**:
   `tsub` では入力カードを中央寄せ (`PlaceHorizontal`) にしているため、左右の余白（マージン）を考慮します。
   ```go
   leftMargin := (m.width - cardWidth) / 2
   // 1 (基準) + 左マージン + 左ボーダー(1: ┃) + 左パディング(1) + プロンプト幅 + テキストエリア内文字幅
   promptWidth := runewidth.StringWidth(m.textarea.Prompt)
   textWidth := m.textarea.LineInfo().CharOffset
   targetX := 1 + leftMargin + 1 + 1 + promptWidth + textWidth
   ```
   > **Note**: `m.textarea.LineInfo().CharOffset` は `uniseg` / `runewidth` を用いて日本語全角文字（幅2）を正しく計算しているため、日本語混じりのテキストでも正確なカラム位置を指します。

### 3.4 Bubble Tea の致命的な挙動: standardRenderer によるカーソル強制上書き
さらに深い調査により、TUI 側（Bubble Tea v1 の `standard_renderer.go`）に決定的な動作があることが判明しました。

```go
// github.com/charmbracelet/bubbletea/standard_renderer.go より抜粋
func (r *standardRenderer) flush() {
    ...
    // Make sure the cursor is at the start of the last line to keep rendering
    // behavior consistent.
    if r.altScreenActive {
        buf.WriteString(ansi.CursorPosition(0, len(newLines)))
    }
    _, _ = r.out.Write(buf.Bytes())
}
```

- Bubble Tea は毎フレームを描画する際、**`View()` が返した文字列をターミナルに書き出した直後に、自動的に `ansi.CursorPosition(0, len(newLines))`（最終行のカラム 0、つまりフッター行の先頭）を強制付与して出力する** 仕様になっていました。
- そのため、単に `View()` の戻り値末尾に ANSI エスケープシーケンス（`\x1b[%d;%dH`）を付与しても、**Bubble Tea の内部レンダラーがその直後にフッター行へのカーソル移動コマンドを送り直して上書きしてしまう**ため、カーソルがフッター行へ引き戻されていたのです。

---

## 4. 解決策の設計と実装 (Solution Architecture)

Bubble Tea のレンダラーによる強制カーソル移動を無効化し、真のハードウェアカーソル位置を制御するため、**出力ストリームのラップ（`cursorWriter`）** を実装しました。

### 4.1 出力レイヤーでのカーソル制御 (`cursorWriter`)
Bubble Tea には `tea.WithOutput(io.Writer)` オプションが用意されています。これを利用して `os.Stdout` をラップする `cursorWriter` を作成し、Bubble Tea がフレームを出力した**直後**に希望のカーソル位置シーケンスを発行します。

```go
// main.go
type cursorWriter struct {
	out io.Writer
}

func (w *cursorWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	if err != nil {
		return n, err
	}
	cy, cx, visible := GetCursorPosition()
	if cy > 0 && cx > 0 {
		var seq string
		if visible {
			seq = fmt.Sprintf("\x1b[?25h\x1b[%d;%dH", cy, cx)
		} else {
			seq = fmt.Sprintf("\x1b[?25l\x1b[%d;%dH", cy, cx)
		}
		_, _ = w.out.Write([]byte(seq))
	}
	return n, nil
}
```

これにより、Bubble Tea レンダラーの出力（フッター行への移動）が端末に届いた直後、瞬時に正しいカーソル位置（入力ボックス内または安全マージン行）へ端末カーソルが上書き再配置されます。

### 4.2 入力モード (`ModeInput`)
ユーザーが入力枠にフォーカスしているときは、エディタ枠内の現在入力行・列へハードウェアカーソルを移動させます。

#### 座標の正確な計算
端末の画面座標系（1-indexed: `(row, col) = (1, 1)` が左上）に合わせた計算式：

1. **行位置 (`targetY`)**:
   ```go
   // 1 (基準) + ヘッダー高 + スペーサー + タイトル高 + 枠線上ボーダー(1) + テキストエリア内行オフセット
   targetY := 1 + headerHeight + spacer1 + editorTitleHeight + 1 + m.textarea.LineInfo().RowOffset
   ```
2. **列位置 (`targetX`)**:
   `tsub` では入力カードを中央寄せ (`PlaceHorizontal`) にしているため、左右の余白（マージン）を考慮します。
   ```go
   leftMargin := (m.width - cardWidth) / 2
   // 1 (基準) + 左マージン + 左ボーダー(1: ┃) + 左パディング(1) + プロンプト幅 + テキストエリア内文字幅
   promptWidth := runewidth.StringWidth(m.textarea.Prompt)
   textWidth := m.textarea.LineInfo().CharOffset
   targetX := 1 + leftMargin + 1 + 1 + promptWidth + textWidth
   ```
   > **Note**: `m.textarea.LineInfo().CharOffset` は `uniseg` / `runewidth` を用いて日本語全角文字（幅2）を正しく計算しているため、日本語混じりのテキストでも正確なカラム位置を指します。

3. **カーソル同期**:
   `View()` 内で `SetCursorPosition(targetY, targetX, true)` を呼び出すことで、`cursorWriter` 経由で毎フレームこの位置へ端末カーソルが同期されます。

これにより、`uim-fep` がカーソル位置を問い合わせた際に入力ボックスの現在入力位置が返るため、**入力枠内のその場（On-The-Spot）に未確定文字列が綺麗に描画**されます。

---

### 4.2 閲覧モード (`ModeView`)
閲覧モードではタイムラインをスクロール（`j` / `k` 等）して閲覧するため、文字入力は行われません。

1. カーソルを非表示化（`\x1b[?25l`）する。
2. さらに、万一 IME が有効なままキー入力が行われた場合に備え、カーソルを**フッターより下の安全なマージン行（`m.height + 1`）へ退避**させる：
   ```go
   parkRow := m.height + 1
   return fullView + fmt.Sprintf("\x1b[?25l\x1b[%d;1H", parkRow)
   ```

これにより、閲覧中にキーを押してもフッターの表示が上書き破壊される事故を完全に防止します。

---

## 5. 設計上の重要なポイントと教訓 (Key Learnings)

1. **GUI と CUI (コンソール) での IME 動作の違い**:
   - GUI（GTK/Qt/Windows/macOS）では OS やウィンドウマネージャ経由で入力エリアの矩形（Rect）を通知しますが、CUI/コンソール環境では **VT100/ANSI のハードウェアカーソル位置（CSI 6 n）が唯一の接点**となります。
2. **TUI 仮想カーソルとハードウェアカーソルの分離**:
   - `bubbles/textarea` や多くの TUI コンポーネントは、端末のハードウェアカーソルを隠し、文字属性（反転表示や `_` など）で「仮想カーソル」を描画することが主流です。
   - しかし、`uim-fep` や `fbterm` などのコンソール FEP 環境では、**ハードウェアカーソルを本物の入力位置に同期させておかないと、IME が描画位置を見失う**という罠があります。
3. **下部ステータスラインと画面行数マージン**:
   - `uim-fep` は最下行（`ws_row - 1`）を自身のステータス行（`anthy-utf8[AnあR]`）として使用します。
   - そのため、TUI 側は `WindowSizeMsg` で得られる画面行数から 2〜3 行のマージン（`TSUB_BOTTOM_MARGIN`）を差し引いて描画高さを決める必要があります。
   - **「高さを引くだけでは不十分で、カーソル位置の制御とセットで初めて完璧に機能する」** というのが今回の最も重要な知見です。
