import type { ReactElement } from 'react';

export default function App(): ReactElement {
  return (
    <main>
      <h1>ETHER Admin</h1>
      <p>Phase 01 骨架，尚無功能。</p>
      <p>
        本頁僅為骨架佔位頁，不提供任何功能，不呼叫任何 API，沒有按鈕、沒有假資料、沒有假儀表板數字。
      </p>
      <p>
        管理員認證與權限是後續階段工作，本階段不實作；管理員權限絕不由前端判斷，以後端強制為準。
      </p>
      <p>
        相關文件：SPEC-SOURCE.md §19（Web
        管理後台）、docs/PLATFORM_MATRIX.md §1.4（Admin：表格／篩選／圖表／審核，獨立認證與權限）、docs/ROADMAP.md
        Phase 01 與 §3.11（管理後台分工）、docs/brand/DESIGN_TOKENS.md（色彩 token 唯一來源）。
      </p>
      <p>TODO：後續功能依階段推進，本階段不實作。字體選用待 Phase 02 定稿，本頁不預寫字體決定。</p>
    </main>
  );
}
