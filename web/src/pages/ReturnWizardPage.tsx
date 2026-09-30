import { useParams, useSearchParams, useNavigate } from 'react-router-dom'
import { useReturn } from '@/hooks/useReturn'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Step1Identity } from './wizard-steps/Step1Identity'
import { Step2Income } from './wizard-steps/Step2Income'
import { Step3Reliefs } from './wizard-steps/Step3Reliefs'
import { Step4Assess } from './wizard-steps/Step4Assess'
import { Step5Validate } from './wizard-steps/Step5Validate'
import { Step6Sign } from './wizard-steps/Step6Sign'
import { Step7Pack } from './wizard-steps/Step7Pack'

export function ReturnWizardPage() {
  const { returnId = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const step = parseInt(params.get('step') ?? '1', 10)

  const { data: proj, isLoading, isError } = useReturn(returnId)

  function goTo(n: number) {
    setParams({ step: String(n) })
  }

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-neutral-50">
        <LoadingSpinner className="h-8 w-8" />
      </div>
    )
  }

  if (isError || !proj) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-neutral-50 px-4">
        <p className="text-neutral-500 text-center">Return not found or access denied.</p>
      </div>
    )
  }

  switch (step) {
    case 1: return (
      <Step1Identity
        returnId={returnId}
        onNext={() => goTo(2)}
      />
    )
    case 2: return (
      <Step2Income
        returnId={returnId}
        proj={proj}
        onNext={() => goTo(3)}
      />
    )
    case 3: return (
      <Step3Reliefs
        returnId={returnId}
        proj={proj}
        onNext={() => goTo(4)}
      />
    )
    case 4: return (
      <Step4Assess
        returnId={returnId}
        proj={proj}
        onNext={() => goTo(5)}
      />
    )
    case 5: return (
      <Step5Validate
        returnId={returnId}
        proj={proj}
        onNext={() => goTo(6)}
      />
    )
    case 6: return (
      <Step6Sign
        returnId={returnId}
        proj={proj}
        onNext={() => goTo(7)}
      />
    )
    case 7: return (
      <Step7Pack
        returnId={returnId}
        proj={proj}
      />
    )
    default:
      navigate(`/returns/${returnId}/file?step=1`, { replace: true })
      return null
  }
}
