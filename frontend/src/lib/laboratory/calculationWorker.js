import { evaluateQuick, evaluateWorkbook } from './calculationEngine.mjs'
self.onmessage = ({ data }) => {
  try {
    self.postMessage({
      id: data.id,
      result:
        data.mode === 'quick'
          ? evaluateQuick(data.expression)
          : evaluateWorkbook(data.content),
    })
  } catch (e) {
    self.postMessage({
      id: data.id,
      error: { code: e.code || '#VALUE!', message: e.message },
    })
  }
}
