import { Route, Routes } from 'react-router'
import { AccessibilityPage } from '@/pages/AccessibilityPage'
import { CouncilAdminPage } from '@/pages/CouncilAdminPage'
import { CouncilSettingsPage } from '@/pages/CouncilSettingsPage'
import { CheckinPrintPage } from '@/pages/CheckinPrintPage'
import { CouncilPage } from '@/pages/CouncilPage'
import { EntrancePage } from '@/pages/EntrancePage'
import { HomePage } from '@/pages/HomePage'
import { FacilityEditorPage } from '@/pages/FacilityEditorPage'
import { FacilityPage } from '@/pages/FacilityPage'
import { InvitePage } from '@/pages/InvitePage'
import { MyBookingsPage } from '@/pages/MyBookingsPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { ReaderPage } from '@/pages/ReaderPage'
import { PlatformAdmin } from '@/pages/PlatformAdmin'
import { SigningIn } from '@/pages/SigningIn'

function App() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/callback" element={<SigningIn />} />
      <Route path="/admin" element={<PlatformAdmin />} />
      <Route path="/perfil" element={<ProfilePage />} />
      <Route path="/lector" element={<ReaderPage />} />
      <Route path="/reservas" element={<MyBookingsPage />} />
      <Route path="/accesibilidade" element={<AccessibilityPage />} />
      <Route path="/invitacion/:token" element={<InvitePage />} />
      <Route path="/:slug" element={<CouncilPage />} />
      <Route path="/:slug/admin" element={<CouncilAdminPage />} />
      <Route path="/:slug/admin/axustes" element={<CouncilSettingsPage />} />
      <Route path="/:slug/admin/facilities/:id" element={<FacilityEditorPage />} />
      <Route path="/:slug/admin/facilities/:id/imprimir" element={<CheckinPrintPage />} />
      <Route path="/:slug/:facility" element={<FacilityPage />} />
      <Route path="/:slug/:facility/entrada" element={<EntrancePage />} />
    </Routes>
  )
}

export default App
