import { Route, Routes } from 'react-router'
import { CouncilAdminPage } from '@/pages/CouncilAdminPage'
import { CouncilPage } from '@/pages/CouncilPage'
import { CouncilPicker } from '@/pages/CouncilPicker'
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
      <Route path="/" element={<CouncilPicker />} />
      <Route path="/callback" element={<SigningIn />} />
      <Route path="/admin" element={<PlatformAdmin />} />
      <Route path="/perfil" element={<ProfilePage />} />
      <Route path="/lector" element={<ReaderPage />} />
      <Route path="/reservas" element={<MyBookingsPage />} />
      <Route path="/invitacion/:token" element={<InvitePage />} />
      <Route path="/:slug" element={<CouncilPage />} />
      <Route path="/:slug/admin" element={<CouncilAdminPage />} />
      <Route path="/:slug/admin/facilities/:id" element={<FacilityEditorPage />} />
      <Route path="/:slug/:facility" element={<FacilityPage />} />
    </Routes>
  )
}

export default App
