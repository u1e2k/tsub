# Linux コンソール・uim-fep (Anthy) 環境における Preedit 重複描画問題の調査報告と解決設計

## 1. 概要 (Executive Summary)

Raspberry Pi 等の小型ディスプレイ（画面行数 12〜14 行程度のフレームバッファ / Linux 仮想コンソール）環境において、日本語入力フロントエンドプロセッサである `uim-fep` (Anthy-utf8) を介して `tsub` を実行した際、**日本語の未確定文字列（Preedit / かな漢字変換中の文字列）が入力ボックスではなく最下部のフッター行（`[入力] Enter: 投稿 ... 終了`）の上に直接上書き描画される**という現象が発生しました。

さらに未確定文字列が長くなると、端末の行末折り返しや改行処理によって画面全体が上スクロールし、ヘッダーや入力欄自体が画面外（上部）へ押し出されてしまう問題も併発していました。

本ドキュメントでは、`uim-fep` のソースコード解析によって判明した描画メカニズム、TUI フレームワーク（Bubble Tea / Lipgloss）との相互作用による根本原因、試行錯誤（入力欄へのカーソル同期 vs ステータスバー移動）の経緯、および完全な解消に至る設計と実装の詳細を後学のために記録します。

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

従来の `tsub` の描画順序：
```
1. HeaderBar       (行 1)
2. EditorBox       (行 2〜5)
3. TimelineView    (行 6〜9)
4. FooterBar       (行 10)  ← [入力] Enter: 投稿 ... 終了
```

### 3.3 Bubble Tea の致命的な挙動: standardRenderer によるカーソル強制上書き
さらに深い調査により、Bubble Tea v1 の `standard_renderer.go` に決定的な動作があることが判明しました。

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

- Bubble Tea は毎フレームを描画する際、**`View()` が返した文字列をターミナルに書き出した直後に、自動的に `ansi.CursorPosition(0, len(newLines))`（最終描画行の先頭）を強制付与して出力する** 仕様になっていました。
- そのため、`tsub` の描画末尾（フッター行）に常にハードウェアカーソルが引き戻されていました。
- `uim-fep` はカーソル位置（＝フッター行）を取得し、**フッター行の上から未確定文字列を直接上書き描画**していたのです。

---

## 4. 解決アプローチの比較と検証 (Trial & Error)

### アプローチ A: 入力ボックス内へハードウェアカーソルを移動させる方式 (v0.1.18)
Bubble Tea の出力ストリームを `cursorWriter` でラップし、フレーム描画直後にエディタ枠内の現在カーソル位置へ `\x1b[%d;%dH` を発行する方式を試行。

- **結果**:
  確かにカーソルは入力枠内に移動し、未確定文字列も入力枠内に現れるようになった。
- **課題**:
  GUI のような独立したフロートウィンドウやインラインインプット機構を持たない CUI / `fbterm` 環境では、`uim-fep` がターミナルの文字セルに直接上書きするため、枠線や周辺文字を破壊したり、入力途中の長い文が不自然に入力枠を突き破るなど、小型画面での視認性・操作性に難があることが判明。

### アプローチ B: 最下部余白を Preedit 専用とし、その直上にステータスラインを配置する方式 (v0.1.23)
ステータスラインを Preedit 行の直上に置くことで文字衝突は回避できたものの、依然として以下の課題が残りました：
- 入力欄上の「💭 いまどうしてる？ (Enter: 投稿)」とステータスバーの「Enter: 投稿」で同じキー案内が重複して表示される。
- タイトル行とステータスバー行で縦に計2行消費し、小型画面でのタイムライン表示領域が圧迫される。

### アプローチ C: ヘッダーへの情報統合＆入力カードのシンプル化 (採用: v0.1.24)
ユーザーからの鋭い着眼点により、UI レイアウトを究極に洗練：
1. **ステータス情報（モードバッジ・操作キーガイド）をヘッダーの空きスペース（右側）へ統合**。
2. 重複していた入力枠のタイトル行（`💭 いまどうしてる？ ...`）を完全撤廃し、入力カード単体として表示。
3. 最下部のステータスバー行も不要となったため廃止。浮いた **縦 2 行分をタイムライン表示領域へ還元**。
4. ハードウェアカーソルは引き続き `Row m.height + 1` の余白行に Park させ、`uim-fep` の Preedit 文字列とステータス行を最下部に独立確保。

#### 最終レイアウト構成:
```text
┌────────────────────────────────────────────────────────────────────────┐
│ tsub  2026-09-19 (Sat) | 10件 [入力] [Anあ] Enter: 投稿  Esc: 閲覧  Ctrl+C: 終了 │ (Header: 状態・IME・操作ガイド統合)
│                                                                        │
│        ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓         │ (EditorCard: タイトル行廃止でコンパクト！)
│        ┃                                                     ┃         │
│        ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛         │
│                                                                        │
│ 20:21 テスト                                                           │ (Timeline: 広々表示！)
│ 20:01 続きから読み込み...                                              │
│ 19:58 あああああああああ                                               │
│ 19:57 え、すご。                                                       │
│                                                                        │
│ 日本語の入力途中文字がここに綺麗に表示される                           │ (uim-fep Preedit 行: Row m.height+1)
│ anthy-utf8[AnあR]                                                      │ (uim-fep Status 行: 最下行)
└────────────────────────────────────────────────────────────────────────┘
```

---

## 5. 実装の詳細 (Implementation)

### 5.1 出力レイヤーでの確実なカーソル配置 (`cursorWriter`)
Bubble Tea の `flush()` によるカーソル移動を確実に上書きするため、`tea.WithOutput` でカスタム Writer を導入：

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

> **重要 (ハマりどころ)**:
> Bubble Tea は標準出力が `term.File`（`io.ReadWriteCloser` かつ `Fd() uintptr` を実装）でない場合、端末サイズ取得に失敗し `WindowSizeMsg` が発火せず「起動中...」でハングします。
> `cursorWriter` は必ず `Read`, `Close`, `Fd()` を委譲実装する必要があります。

### 5.2 ステータスバー配置とカーソルパーク (`model.go`)
```go
// model.go View()
sections = append(sections, editorRendered)
sections = append(sections, timelineRendered)
sections = append(sections, statusBar) // Preedit 行の直上に配置

// uim-fep (Anthy) の未確定文字列描画用として、最下部余白行 (m.height + 1) にカーソルを退避
parkRow := m.height + 1
if m.mode == ModeInput {
    SetCursorPosition(parkRow, 1, true)
    return fullView + fmt.Sprintf("\x1b[?25h\x1b[%d;1H", parkRow)
}
SetCursorPosition(parkRow, 1, false)
return fullView + fmt.Sprintf("\x1b[?25l\x1b[%d;1H", parkRow)
```

---

## 6. 設計上の重要なポイントと教訓 (Key Learnings)

1. **コンソール IME (FEP) と CUI アプリの共存モデル**:
   - GUI と異なり、VT100 端末のハードウェアカーソルは画面上に**たった1つ**しか存在しません。
   - アプリケーションが画面最下部まで目一杯 TUI コンポーネントを詰め込むと、FEP は既存の表示要素を上書きせざるを得なくなります。
   - **「TUI の描画領域を意図的に 2〜3 行空け、そこにハードウェアカーソルを待機させる」** ことで、FEP 専用の独立した表示スペースを確保するのが Linux コンソール環境における最も堅牢で安定した設計パターンです。
2. **ステータスバーの配置柔軟性**:
   - 「ステータスバーや操作ヘルプは常に最下部にあるもの」という固定観念を捨て、エディタカードの直下に置くことで、エディタと操作ガイドの視線移動が減少し、かつ最下部の IME 領域との衝突を完全にゼロにできます。
