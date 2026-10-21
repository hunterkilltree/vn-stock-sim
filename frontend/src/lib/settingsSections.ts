// Settings subnav from design/screens/Settings-AI.dc.html's subRaw --
// same labels and order. "ai" (Phase G shell, Phase H form) and
// "paper-account" (phase-risk-rules.md slice 1) are built; the rest carry
// the design's own WILL badge (phase-g.md decision 11).
export type SettingsSection = { slug: string; label: string; built: boolean };

export const settingsSections: SettingsSection[] = [
  { slug: "account", label: "Tài khoản", built: false },
  { slug: "ai", label: "Mô hình AI", built: true },
  { slug: "market-data", label: "Dữ liệu thị trường", built: false },
  { slug: "paper-account", label: "Tài khoản giấy", built: true },
  { slug: "replay", label: "Replay và chấm điểm", built: false },
  { slug: "notifications", label: "Thông báo", built: false },
  { slug: "appearance", label: "Giao diện", built: false },
  { slug: "privacy", label: "Quyền riêng tư", built: false },
];
