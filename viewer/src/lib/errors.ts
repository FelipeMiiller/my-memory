/** Converte um erro desconhecido em uma mensagem exibível. */
export function describeUnknownError(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === "string") return error
  try {
    return JSON.stringify(error)
  } catch {
    return String(error)
  }
}
