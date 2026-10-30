import React, { useState } from 'react'
import {
  LaboratoryProvider,
  LaboratoryTrigger,
} from '../../src/components/laboratory/LaboratoryProvider'
import Laboratory from '../../src/pages/Laboratory'
import FiscalSimulation from '../../src/components/FiscalSimulation'
export default function FiscalLaboratoryHarness() {
  const [mode, setMode] = useState('lab')
  return (
    <LaboratoryProvider owner={987654321} onNavigate={() => setMode('lab')}>
      <h1>Parcours TEST</h1>
      <button onClick={() => setMode(mode === 'lab' ? 'fiscal' : 'lab')}>
        Changer de parcours
      </button>
      <LaboratoryTrigger />
      {mode === 'lab' ? <Laboratory /> : <FiscalSimulation />}
    </LaboratoryProvider>
  )
}
