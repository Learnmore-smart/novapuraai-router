/// <reference types="@rsbuild/core/types" />

declare module '@visactor/react-vchart' {
  export const VChart: React.ComponentType<Record<string, unknown>>
}

declare module '@visactor/vchart-semi-theme' {
  export const initVChartSemiTheme: (opts?: Record<string, unknown>) => void
}

declare module '*.md?raw' {
  const content: string
  export default content
}

declare module '*.md' {
  const content: string
  export default content
}
