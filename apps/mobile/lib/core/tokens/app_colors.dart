import 'package:flutter/material.dart';

// ETHER 品牌色。
//
// 唯一來源：docs/brand/DESIGN_TOKENS.md §1.1（品牌企劃書 §8）。
// 本檔案只寫以下 8 個已定色碼，逐字核對過，不得增減、不得改值。
// 其餘 token（surface-elevated、border、success／warning／error／info、
// 字體、間距、圓角、陰影、動效、圖示、z-index 等）一律待 Phase 02 定稿，
// 以下僅以 TODO 註解標示，絕對不得編造色碼或數值。

/// 已定的 8 個品牌色（DESIGN_TOKENS.md §1.1）。
class AppColors {
  const AppColors._();

  /// Ether Midnight：品牌深色背景。
  static const Color etherMidnight = Color(0xFF101923);

  /// Deep Nebula：深層介面。
  static const Color deepNebula = Color(0xFF182A3A);

  /// Ethereal Blue：品牌識別色。
  static const Color etherealBlue = Color(0xFF86B9D2);

  /// Silver Mist：輔助文字。
  static const Color silverMist = Color(0xFFC7D5DE);

  /// Pure Light：明亮背景。
  static const Color pureLight = Color(0xFFF4F7F9);

  /// Aurora Cyan：互動與狀態。
  static const Color auroraCyan = Color(0xFF65D4D6);

  /// Soft Violet：AI 與特殊功能。
  static const Color softViolet = Color(0xFFA99ACB);

  /// Warm Horizon：情感與記憶內容。
  static const Color warmHorizon = Color(0xFFD9B98C);
}

// TODO(Phase 02): surface-elevated、border、success／warning／error／info
//   等色彩 token，待 Phase 02 設計系統定稿，不得編造。
// TODO(Phase 02): 字體 token（font.family／scale／weight／line-height），
//   待 Phase 02 定稿，不得編造。
// TODO(Phase 02): spacing／radius／border／shadow／elevation／motion／
//   icon／z-index token，待 Phase 02 定稿，不得編造。
