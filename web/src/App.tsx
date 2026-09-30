import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthContext, useAuthState } from '@/hooks/useAuth'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { NewReturnPage } from '@/pages/NewReturnPage'
import { ReturnWizardPage } from '@/pages/ReturnWizardPage'
import { OnboardingPage } from '@/pages/OnboardingPage'
import { OptimizerPage } from '@/pages/OptimizerPage'
import { AuditSupportPage } from '@/pages/AuditSupportPage'
import { PricingPage } from '@/pages/PricingPage'
import { TaxProMarketplace } from '@/pages/TaxProMarketplace'
import { CryptoCGTPage } from '@/pages/CryptoCGTPage'
import { DeductionsWizardPage } from '@/pages/DeductionsWizardPage'
import { ReadyCheckPage } from '@/pages/ReadyCheckPage'
import { EFilePage } from '@/pages/EFilePage'
import { PaymentPage } from '@/pages/PaymentPage'
import { TCCPage } from '@/pages/TCCPage'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
})

export function App() {
  const auth = useAuthState()

  return (
    <AuthContext.Provider value={auth}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            {/* Public */}
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />

            {/* Onboarding */}
            <Route
              path="/onboarding"
              element={auth.isAuthenticated ? <OnboardingPage /> : <Navigate to="/login" replace />}
            />

            {/* Protected */}
            <Route
              path="/"
              element={auth.isAuthenticated ? <Navigate to="/dashboard" replace /> : <Navigate to="/login" replace />}
            />
            <Route
              path="/dashboard"
              element={auth.isAuthenticated ? <DashboardPage /> : <Navigate to="/login" replace />}
            />
            <Route
              path="/returns/new"
              element={auth.isAuthenticated ? <NewReturnPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/returns/:returnId/file"
              element={auth.isAuthenticated ? <ReturnWizardPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/optimizer"
              element={auth.isAuthenticated ? <OptimizerPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/help/audit"
              element={auth.isAuthenticated ? <AuditSupportPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/pricing"
              element={auth.isAuthenticated ? <PricingPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/experts"
              element={auth.isAuthenticated ? <TaxProMarketplace /> : <Navigate to="/login" replace />}
            />
            <Route path="/crypto"
              element={auth.isAuthenticated ? <CryptoCGTPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/deductions"
              element={auth.isAuthenticated ? <DeductionsWizardPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/ready-check"
              element={auth.isAuthenticated ? <ReadyCheckPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/efile"
              element={auth.isAuthenticated ? <EFilePage /> : <Navigate to="/login" replace />}
            />
            <Route path="/payment"
              element={auth.isAuthenticated ? <PaymentPage /> : <Navigate to="/login" replace />}
            />
            <Route path="/tcc"
              element={auth.isAuthenticated ? <TCCPage /> : <Navigate to="/login" replace />}
            />

            {/* Fallback */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </AuthContext.Provider>
  )
}
