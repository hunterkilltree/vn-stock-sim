import AISettingsForm from "@/components/AISettingsForm";

export const metadata = { title: "Mô hình AI — Cài đặt — VN Stock Sim" };

// Settings-AI.dc.html, built in Phase H (phase-h.md). All values live in
// this browser's localStorage, so the whole form is a Client Component.
export default function SettingsAIPage() {
  return <AISettingsForm />;
}
