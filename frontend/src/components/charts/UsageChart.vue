<script setup lang="ts">
/**
 * UsageChart.vue - Reusable time-series line chart component
 *
 * Uses Chart.js with vue-chartjs for rendering interactive time-series charts.
 * Supports multiple datasets, configurable colors, and automatic scaling.
 */
import { computed, ref, watch, onMounted } from 'vue'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler,
  type ChartData,
  type ChartOptions
} from 'chart.js'
import annotationPlugin from 'chartjs-plugin-annotation'
import 'chartjs-adapter-date-fns'

// Register Chart.js components (required for tree-shaking)
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler,
  annotationPlugin
)

// Props interface for type safety
export interface DataPoint {
  x: Date | string | number
  y: number
}

export interface Dataset {
  label: string
  data: DataPoint[] | number[]
  borderColor?: string
  backgroundColor?: string
  fill?: boolean
  tension?: number
  pointRadius?: number
  pointHoverRadius?: number
}

export interface PeakAnnotation {
  /** X-coordinate for the peak point (Date, string, or index) */
  x: Date | string | number
  /** Y-coordinate value for the peak point */
  y: number
  /** Label to display on the annotation */
  label?: string
  /** Additional value to show in tooltip (e.g., peak concurrent tunnels) */
  tooltipValue?: number | string
}

const props = withDefaults(defineProps<{
  /** Array of datasets to display */
  datasets: Dataset[]
  /** Labels for x-axis (used when datasets contain number[] instead of DataPoint[]) */
  labels?: string[]
  /** Chart title */
  title?: string
  /** Y-axis label */
  yAxisLabel?: string
  /** X-axis label */
  xAxisLabel?: string
  /** Use time scale for x-axis */
  useTimeScale?: boolean
  /** Time unit for time scale ('hour', 'day', 'week', 'month') */
  timeUnit?: 'hour' | 'day' | 'week' | 'month'
  /** Display format for time labels */
  timeDisplayFormat?: string
  /** Chart height in pixels */
  height?: number
  /** Enable area fill under line */
  fill?: boolean
  /** Enable data decimation for large datasets */
  enableDecimation?: boolean
  /** Format function for y-axis ticks */
  yAxisFormatter?: (value: number) => string
  /** Format function for tooltip values */
  tooltipFormatter?: (value: number) => string
  /** Show legend */
  showLegend?: boolean
  /** Loading state */
  loading?: boolean
  /** Peak annotations to highlight on the chart */
  peakAnnotations?: PeakAnnotation[]
  /** Show peak annotations (auto-detect max values if peakAnnotations not provided) */
  showPeakAnnotations?: boolean
}>(), {
  labels: () => [],
  title: '',
  yAxisLabel: '',
  xAxisLabel: '',
  useTimeScale: false,
  timeUnit: 'day',
  timeDisplayFormat: 'MMM d',
  height: 300,
  fill: false,
  enableDecimation: true,
  showLegend: true,
  loading: false,
  peakAnnotations: () => [],
  showPeakAnnotations: false
})

// Default color palette matching design system
const defaultColors = [
  { border: 'rgb(136, 91, 247)', background: 'rgba(136, 91, 247, 0.1)' }, // primary/purple
  { border: 'rgb(45, 212, 191)', background: 'rgba(45, 212, 191, 0.1)' }, // secondary/teal
  { border: 'rgb(251, 191, 36)', background: 'rgba(251, 191, 36, 0.1)' }, // amber
  { border: 'rgb(59, 130, 246)', background: 'rgba(59, 130, 246, 0.1)' }, // blue
  { border: 'rgb(239, 68, 68)', background: 'rgba(239, 68, 68, 0.1)' }, // red
]

// Transform datasets with default styling
const chartData = computed(() => {
  const transformedDatasets = props.datasets.map((dataset, index) => {
    const colorIndex = index % defaultColors.length
    const colorSet = defaultColors[colorIndex]!
    return {
      label: dataset.label,
      data: dataset.data as unknown[],
      borderColor: dataset.borderColor || colorSet.border,
      backgroundColor: dataset.backgroundColor || colorSet.background,
      fill: dataset.fill ?? props.fill,
      tension: dataset.tension ?? 0.3,
      pointRadius: dataset.pointRadius ?? 3,
      pointHoverRadius: dataset.pointHoverRadius ?? 6,
      borderWidth: 2
    }
  })

  return {
    labels: props.labels,
    datasets: transformedDatasets
  } as ChartData<'line'>
})

// Compute peak annotations from provided data or auto-detect from datasets
const computedPeakAnnotations = computed(() => {
  // If explicit peak annotations are provided, use them
  if (props.peakAnnotations && props.peakAnnotations.length > 0) {
    return props.peakAnnotations
  }

  // If showPeakAnnotations is enabled, auto-detect max values from each dataset
  if (props.showPeakAnnotations && props.datasets.length > 0) {
    const peaks: PeakAnnotation[] = []

    props.datasets.forEach((dataset) => {
      const dataPoints = dataset.data
      if (!dataPoints || dataPoints.length === 0) return

      // Find the maximum value point - store x and y separately to avoid type narrowing issues
      let maxX: Date | string | number | null = null
      let maxY: number | null = null
      let maxValue = -Infinity

      dataPoints.forEach((point) => {
        // Handle both DataPoint objects and plain numbers
        if (typeof point === 'object' && point !== null && 'y' in point) {
          const dataPoint = point as DataPoint
          if (dataPoint.y > maxValue) {
            maxValue = dataPoint.y
            maxX = dataPoint.x
            maxY = dataPoint.y
          }
        } else if (typeof point === 'number') {
          if (point > maxValue) {
            maxValue = point
          }
        }
      })

      if (maxX !== null && maxY !== null && maxValue > 0) {
        peaks.push({
          x: maxX,
          y: maxY,
          label: `Peak: ${dataset.label}`,
          tooltipValue: maxValue
        })
      }
    })

    return peaks
  }

  return []
})

// Generate Chart.js annotation objects from peak annotations
const chartAnnotations = computed(() => {
  const annotations: Record<string, unknown> = {}

  computedPeakAnnotations.value.forEach((peak, index) => {
    // Point annotation to highlight the peak
    annotations[`peak-point-${index}`] = {
      type: 'point',
      xValue: peak.x,
      yValue: peak.y,
      xScaleID: 'x',
      yScaleID: 'y',
      backgroundColor: 'rgba(251, 191, 36, 0.3)', // amber with transparency
      borderColor: 'rgb(251, 191, 36)', // amber
      borderWidth: 2,
      radius: 8,
      pointStyle: 'circle',
      drawTime: 'afterDatasetsDraw'
    }

    // Label annotation to show "Peak" text
    if (peak.label) {
      annotations[`peak-label-${index}`] = {
        type: 'label',
        xValue: peak.x,
        yValue: peak.y,
        xScaleID: 'x',
        yScaleID: 'y',
        content: peak.tooltipValue !== undefined
          ? `⬆ Peak${peak.tooltipValue ? `: ${peak.tooltipValue}` : ''}`
          : '⬆ Peak',
        backgroundColor: 'rgba(17, 24, 39, 0.9)',
        color: 'rgb(251, 191, 36)',
        font: {
          size: 11,
          weight: 'bold',
          family: "'Inter', sans-serif"
        },
        padding: { top: 4, bottom: 4, left: 6, right: 6 },
        borderRadius: 4,
        yAdjust: -20,
        drawTime: 'afterDatasetsDraw'
      }
    }
  })

  return annotations
})

// Chart options with responsive configuration
const chartOptions = computed<ChartOptions<'line'>>(() => {
  const options: ChartOptions<'line'> = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false
    },
    plugins: {
      legend: {
        display: props.showLegend && props.datasets.length > 1,
        position: 'top',
        labels: {
          color: 'rgb(156, 163, 175)', // text-secondary
          usePointStyle: true,
          padding: 16,
          font: {
            family: "'Inter', sans-serif",
            size: 12
          }
        }
      },
      title: {
        display: !!props.title,
        text: props.title,
        color: 'rgb(249, 250, 251)', // text-primary
        font: {
          family: "'Space Grotesk', sans-serif",
          size: 16,
          weight: 600
        },
        padding: {
          bottom: 16
        }
      },
      tooltip: {
        backgroundColor: 'rgba(17, 24, 39, 0.95)',
        titleColor: 'rgb(249, 250, 251)',
        bodyColor: 'rgb(156, 163, 175)',
        borderColor: 'rgba(75, 85, 99, 0.3)',
        borderWidth: 1,
        padding: 12,
        cornerRadius: 8,
        titleFont: {
          family: "'Inter', sans-serif",
          size: 13,
          weight: 600
        },
        bodyFont: {
          family: "'Inter', sans-serif",
          size: 12
        },
        callbacks: {
          label: (context) => {
            const value = context.parsed.y
            if (value === null || value === undefined) return ''
            const formattedValue = props.tooltipFormatter
              ? props.tooltipFormatter(value)
              : value.toLocaleString()
            return `${context.dataset.label}: ${formattedValue}`
          },
          afterBody: (tooltipItems) => {
            // Check if this point is a peak and add peak info to tooltip
            if (tooltipItems.length === 0 || computedPeakAnnotations.value.length === 0) {
              return ''
            }

            const item = tooltipItems[0]
            if (!item) return ''

            const xValue = item.parsed.x
            const yValue = item.parsed.y

            // Find if this point matches any peak annotation
            const matchingPeak = computedPeakAnnotations.value.find(peak => {
              const peakX = peak.x instanceof Date ? peak.x.getTime() : peak.x
              let itemX: number | string
              if (typeof xValue === 'number') {
                itemX = xValue
              } else if (xValue !== null && xValue !== undefined) {
                itemX = new Date(String(xValue)).getTime()
              } else {
                return false
              }
              return peakX === itemX && peak.y === yValue
            })

            if (matchingPeak && matchingPeak.tooltipValue !== undefined) {
              return `\n⭐ Peak Value: ${matchingPeak.tooltipValue}`
            }

            return ''
          }
        }
      }
    },
    scales: {
      x: props.useTimeScale
        ? {
            type: 'time',
            time: {
              unit: props.timeUnit,
              displayFormats: {
                hour: 'HH:mm',
                day: props.timeDisplayFormat,
                week: 'MMM d',
                month: 'MMM yyyy'
              }
            },
            title: {
              display: !!props.xAxisLabel,
              text: props.xAxisLabel,
              color: 'rgb(156, 163, 175)'
            },
            grid: {
              color: 'rgba(75, 85, 99, 0.2)',
              drawOnChartArea: true
            },
            ticks: {
              color: 'rgb(156, 163, 175)',
              font: {
                family: "'Inter', sans-serif",
                size: 11
              },
              maxRotation: 0
            }
          }
        : {
            type: 'category',
            title: {
              display: !!props.xAxisLabel,
              text: props.xAxisLabel,
              color: 'rgb(156, 163, 175)'
            },
            grid: {
              color: 'rgba(75, 85, 99, 0.2)',
              drawOnChartArea: true
            },
            ticks: {
              color: 'rgb(156, 163, 175)',
              font: {
                family: "'Inter', sans-serif",
                size: 11
              },
              maxRotation: 45
            }
          },
      y: {
        beginAtZero: true,
        title: {
          display: !!props.yAxisLabel,
          text: props.yAxisLabel,
          color: 'rgb(156, 163, 175)',
          font: {
            family: "'Inter', sans-serif",
            size: 12
          }
        },
        grid: {
          color: 'rgba(75, 85, 99, 0.2)'
        },
        ticks: {
          color: 'rgb(156, 163, 175)',
          font: {
            family: "'Inter', sans-serif",
            size: 11
          },
          callback: (tickValue) => {
            const value = typeof tickValue === 'number' ? tickValue : parseFloat(tickValue)
            return props.yAxisFormatter ? props.yAxisFormatter(value) : value.toLocaleString()
          }
        }
      }
    }
  }

  // Add decimation for large datasets (performance optimization)
  if (props.enableDecimation) {
    options.parsing = false
    options.plugins = {
      ...options.plugins,
      decimation: {
        enabled: true,
        algorithm: 'lttb',
        samples: 500
      }
    }
  }

  // Add peak annotations if enabled
  const annotations = chartAnnotations.value
  if (Object.keys(annotations).length > 0) {
    // Use type assertion since we're building valid annotation objects dynamically
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (options.plugins as any).annotation = {
      annotations
    }
  }

  return options
})

// Expose chart instance for external control if needed
const chartRef = ref<InstanceType<typeof Line> | null>(null)

defineExpose({
  chartRef
})
</script>

<template>
  <div
    class="bg-bg-surface border border-border-subtle rounded-xs p-5 relative overflow-hidden"
    :style="{ minHeight: `${height}px` }"
  >
    <!-- Loading State -->
    <div
      v-if="loading"
      class="absolute inset-0 flex items-center justify-center bg-bg-surface/80 z-10"
    >
      <div class="flex flex-col items-center gap-3">
        <div class="w-8 h-8 border-2 border-accent-primary border-t-transparent rounded-full animate-spin" />
        <span class="text-sm text-text-secondary">Loading chart data...</span>
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-else-if="!datasets.length || datasets.every(d => !d.data.length)"
      class="absolute inset-0 flex items-center justify-center"
    >
      <div class="text-center">
        <div class="text-4xl mb-3 opacity-30">📊</div>
        <p class="text-text-secondary text-sm">No data available</p>
      </div>
    </div>

    <!-- Chart -->
    <div v-else class="w-full h-full" :style="{ height: `${height}px` }">
      <Line
        ref="chartRef"
        :data="chartData"
        :options="chartOptions"
      />
    </div>
  </div>
</template>
