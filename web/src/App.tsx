import { Route, Routes } from 'react-router'
import { LegalPage } from '@/pages/LegalPage'
import { AccessibilityPage } from '@/pages/AccessibilityPage'
import { CouncilAdminPage } from '@/pages/CouncilAdminPage'
import { CouncilSettingsPage } from '@/pages/CouncilSettingsPage'
import { PlatformSettingsPage } from '@/pages/PlatformSettingsPage'
import { WidgetPage } from '@/pages/WidgetPage'
import { CheckinPrintPage } from '@/pages/CheckinPrintPage'
import { CouncilPage } from '@/pages/CouncilPage'
import { EntrancePage } from '@/pages/EntrancePage'
import { HomePage } from '@/pages/HomePage'
import { EventEditorPage } from '@/pages/EventEditorPage'
import { FacilityEditorPage } from '@/pages/FacilityEditorPage'
import { FacilityPage } from '@/pages/FacilityPage'
import { InvitePage } from '@/pages/InvitePage'
import { MyBookingsPage } from '@/pages/MyBookingsPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { ReaderPage } from '@/pages/ReaderPage'
import { PlatformAdmin } from '@/pages/PlatformAdmin'
import { SigningIn } from '@/pages/SigningIn'
import { DataAssistantPage } from '@/pages/DataAssistantPage'
import { DataBubble } from '@/components/DataBubble'

function App() {
  return (
    <>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/callback" element={<SigningIn />} />
        <Route path="/admin" element={<PlatformAdmin />} />
        <Route path="/admin/axustes" element={<PlatformSettingsPage />} />
        <Route path="/datos" element={<DataAssistantPage />} />
        <Route path="/perfil" element={<ProfilePage />} />
        <Route path="/lector" element={<ReaderPage />} />
        <Route path="/reservas" element={<MyBookingsPage />} />
        <Route path="/accesibilidade" element={<AccessibilityPage />} />
        <Route path="/condicions" element={<LegalPage which="terms" />} />
        <Route path="/privacidade" element={<LegalPage which="privacy" />} />
        <Route path="/invitacion/:token" element={<InvitePage />} />
        <Route path="/:slug" element={<CouncilPage />} />
        {/* embedded in a council's website (the proxy lets only these be framed) */}
        <Route path="/:slug/widget" element={<WidgetPage />} />
        <Route path="/:slug/widget/:facility" element={<WidgetPage />} />
        <Route path="/:slug/admin" element={<CouncilAdminPage />} />
        <Route path="/:slug/admin/axustes" element={<CouncilSettingsPage />} />
        <Route path="/:slug/admin/eventos/:id" element={<EventEditorPage />} />
        <Route path="/:slug/admin/facilities/:id" element={<FacilityEditorPage />} />
        <Route path="/:slug/admin/facilities/:id/imprimir" element={<CheckinPrintPage />} />
        <Route path="/:slug/:facility" element={<FacilityPage />} />
        <Route path="/:slug/:facility/entrada" element={<EntrancePage />} />
      </Routes>
      <DataBubble />
    </>
  )
}

export default App
