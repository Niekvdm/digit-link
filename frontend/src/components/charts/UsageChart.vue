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
  Filler
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
  loading: false
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
