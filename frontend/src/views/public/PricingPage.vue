<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { PlanCard } from '@/components/ui'
import type { Plan } from '@/types/api'
import { 
  Check, 
  X, 
  ChevronDown,
  Mail,
  ArrowRight,
  Zap,
  Shield,
  Globe,
  Clock
} from 'lucide-vue-next'

const router = useRouter()
const plans = ref<Plan[]>([])
const loading = ref(true)
const expandedFaq = ref<number | null>(null)

onMounted(async () => {
  try {
    const response = await fetch('/api/plans')
    if (response.ok) {
      const data = await response.json()
      plans.value = data.plans || []
    }
  } catch (e) {
    console.error('Failed to fetch plans:', e)
  } finally {
    loading.value = false
  }
})

const features = [
  { name: 'Secure Tunnels', free: true, pro: true, enterprise: true },
  { name: 'Custom Subdomains', free: true, pro: true, enterprise: true },
  { name: 'TLS Encryption', free: true, pro: true, enterprise: true },
  { name: 'Basic Auth', free: true, pro: true, enterprise: true },
  { name: 'API Key Auth', free: false, pro: true, enterprise: true },
  { name: 'OIDC/OAuth', free: false, pro: true, enterprise: true },
  { name: 'IP Whitelisting', free: false, pro: true, enterprise: true },
  { name: 'Audit Logs', free: false, pro: true, enterprise: true },
  { name: 'Multi-user Access', free: false, pro: true, enterprise: true },
  { name: 'Priority Support', free: false, pro: false, enterprise: true },
  { name: 'Custom SLA', free: false, pro: false, enterprise: true },
  { name: 'Dedicated Infrastructure', free: false, pro: false, enterprise: true },
]

const faqs = [
  {
    question: 'What happens when I reach my bandwidth limit?',
    answer: 'Depending on your plan, you may have a grace period or overage allowance. Free plans have hard limits, while paid plans typically include a grace period to ensure your services stay online while you upgrade.'
  },
  {
    question: 'Can I change plans at any time?',
    answer: 'Yes! You can upgrade or downgrade your plan at any time. Changes take effect immediately, and any unused quota from your current billing period will be prorated.'
  },
  {
    question: 'How does tunnel hour billing work?',
    answer: 'Tunnel hours are counted from when a tunnel connects until it disconnects. If you have a tunnel running 24/7 for a month, that\'s approximately 720 hours. Idle tunnels still count towards your usage.'
  },
  {
    question: 'Is there a free trial for paid plans?',
    answer: 'We offer a 14-day free trial for Pro plans. Contact our sales team for Enterprise trials and custom evaluation periods.'
  },
  {
    question: 'What payment methods do you accept?',
    answer: 'We accept all major credit cards (Visa, Mastercard, American Express) and can arrange invoicing for Enterprise customers.'
  },
]

function toggleFaq(index: number) {
  expandedFaq.value = expandedFaq.value === index ? null : index
}

function goToLogin() {
  router.push({ name: 'login' })
}

function contactSales() {
  window.location.href = 'mailto:sales@example.com?subject=digit-link%20Enterprise%20Inquiry'
}
</script>

<template>
  <div class="min-h-screen bg-bg-deep">
    <!-- Decorative background -->
    <div class="fixed inset-0 pointer-events-none">
      <div class="absolute inset-0 bg-[radial-gradient(ellipse_at_30%_20%,rgba(var(--accent-primary-rgb),0.08)_0%,transparent_50%)]" />
      <div class="absolute inset-0 bg-[radial-gradient(ellipse_at_70%_80%,rgba(var(--accent-secondary-rgb),0.05)_0%,transparent_50%)]" />
    </div>

    <!-- Header -->
    <header class="relative z-10 border-b border-border-subtle bg-bg-surface/80 backdrop-blur-sm">
      <div class="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 border-2 border-accent-primary rounded-sm flex items-center justify-center">
            <div class="w-3.5 h-3.5 rounded bg-accent-primary rotate-45" />
          </div>
          <span class="font-display text-xl font-semibold text-text-primary">digit-link</span>
        </div>
        <button class="btn btn-primary" @click="goToLogin">
          Sign In
          <ArrowRight class="w-4 h-4" />
        </button>
      </div>
    </header>

    <!-- Hero Section -->
    <section class="relative z-10 py-20 px-6">
      <div class="max-w-4xl mx-auto text-center">
        <h1 class="font-display text-5xl md:text-6xl font-bold text-text-primary mb-6 leading-tight">
          Simple, transparent
          <span class="text-accent-primary">pricing</span>
        </h1>
        <p class="text-xl text-text-secondary max-w-2xl mx-auto mb-8">
          Secure tunnel access for your development and production workloads. 
          Start free and scale as you grow.
        </p>
        
        <!-- Value props -->
        <div class="flex flex-wrap justify-center gap-6 text-sm text-text-secondary">
          <div class="flex items-center gap-2">
            <Shield class="w-4 h-4 text-accent-secondary" />
            <span>Enterprise-grade security</span>
          </div>
          <div class="flex items-center gap-2">
            <Globe class="w-4 h-4 text-accent-secondary" />
            <span>Global edge network</span>
          </div>
          <div class="flex items-center gap-2">
            <Zap class="w-4 h-4 text-accent-secondary" />
            <span>99.9% uptime SLA</span>
          </div>
          <div class="flex items-center gap-2">
            <Clock class="w-4 h-4 text-accent-secondary" />
            <span>24/7 support available</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Plans Grid -->
    <section class="relative z-10 pb-20 px-6">
      <div class="max-w-6xl mx-auto">
        <div v-if="loading" class="flex justify-center py-20">
          <div class="w-8 h-8 border-2 border-accent-primary border-t-transparent rounded-full animate-spin" />
        </div>
        
        <div 
          v-else-if="plans.length > 0"
          class="grid gap-6"
          :class="plans.length <= 2 ? 'md:grid-cols-2 max-w-3xl mx-auto' : 'md:grid-cols-2 lg:grid-cols-3'"
        >
          <PlanCard
            v-for="plan in plans"
            :key="plan.id"
            :plan="plan"
          >
            <template #actions>
              <button 
                v-if="plan.name.toLowerCase().includes('free')"
                class="btn btn-secondary w-full"
                @click="goToLogin"
              >
                Get Started Free
              </button>
              <button 
                v-else-if="plan.name.toLowerCase().includes('enterprise')"
                class="btn btn-primary w-full"
                @click="contactSales"
              >
                Contact Sales
              </button>
              <button 
                v-else
                class="btn btn-primary w-full"
                @click="contactSales"
              >
                Start Free Trial
              </button>
            </template>
          </PlanCard>
        </div>

        <div v-else class="text-center py-20 text-text-muted">
          <p>No plans available. Contact us for pricing.</p>
        </div>
      </div>
    </section>

    <!-- Feature Comparison -->
    <section class="relative z-10 py-20 px-6 bg-bg-surface/50 border-y border-border-subtle">
      <div class="max-w-4xl mx-auto">
        <h2 class="font-display text-3xl font-bold text-text-primary text-center mb-12">
          Compare Features
        </h2>
        
        <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
          <table class="w-full">
            <thead>
              <tr class="border-b border-border-subtle bg-bg-elevated">
                <th class="text-left py-4 px-6 text-sm font-semibold text-text-primary">Feature</th>
                <th class="text-center py-4 px-4 text-sm font-semibold text-text-primary w-24">Free</th>
                <th class="text-center py-4 px-4 text-sm font-semibold text-accent-blue w-24">Pro</th>
                <th class="text-center py-4 px-4 text-sm font-semibold text-[rgb(168,85,247)] w-24">Enterprise</th>
              </tr>
            </thead>
            <tbody>
              <tr 
                v-for="(feature, index) in features" 
                :key="feature.name"
                class="border-b border-border-subtle last:border-0"
                :class="index % 2 === 0 ? '' : 'bg-bg-elevated/30'"
              >
                <td class="py-3.5 px-6 text-sm text-text-secondary">{{ feature.name }}</td>
                <td class="text-center py-3.5 px-4">
                  <Check v-if="feature.free" class="w-5 h-5 text-accent-secondary mx-auto" />
                  <X v-else class="w-5 h-5 text-text-muted mx-auto opacity-30" />
                </td>
                <td class="text-center py-3.5 px-4">
                  <Check v-if="feature.pro" class="w-5 h-5 text-accent-blue mx-auto" />
                  <X v-else class="w-5 h-5 text-text-muted mx-auto opacity-30" />
                </td>
                <td class="text-center py-3.5 px-4">
                  <Check v-if="feature.enterprise" class="w-5 h-5 text-[rgb(168,85,247)] mx-auto" />
                  <X v-else class="w-5 h-5 text-text-muted mx-auto opacity-30" />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <!-- FAQ Section -->
    <section class="relative z-10 py-20 px-6">
      <div class="max-w-3xl mx-auto">
        <h2 class="font-display text-3xl font-bold text-text-primary text-center mb-12">
          Frequently Asked Questions
        </h2>
        
        <div class="space-y-3">
          <div 
            v-for="(faq, index) in faqs" 
            :key="index"
            class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden"
          >
            <button 
              class="w-full flex items-center justify-between p-5 text-left cursor-pointer bg-transparent border-none"
              @click="toggleFaq(index)"
            >
              <span class="font-semibold text-text-primary pr-4">{{ faq.question }}</span>
              <ChevronDown 
                class="w-5 h-5 text-text-muted shrink-0 transition-transform duration-200"
                :class="{ 'rotate-180': expandedFaq === index }"
              />
            </button>
            <Transition name="faq">
              <div v-if="expandedFaq === index" class="px-5 pb-5">
                <p class="text-text-secondary text-sm leading-relaxed">{{ faq.answer }}</p>
              </div>
            </Transition>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA Section -->
    <section class="relative z-10 py-20 px-6 bg-gradient-to-b from-bg-surface/50 to-bg-deep border-t border-border-subtle">
      <div class="max-w-3xl mx-auto text-center">
        <h2 class="font-display text-3xl font-bold text-text-primary mb-4">
          Ready to get started?
        </h2>
        <p class="text-text-secondary mb-8">
          Join thousands of developers using digit-link to securely expose their services.
        </p>
        <div class="flex flex-col sm:flex-row items-center justify-center gap-4">
          <button class="btn btn-primary btn-lg" @click="goToLogin">
            Start for Free
            <ArrowRight class="w-5 h-5" />
          </button>
          <button class="btn btn-secondary btn-lg" @click="contactSales">
            <Mail class="w-5 h-5" />
            Contact Sales
          </button>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="relative z-10 py-8 px-6 border-t border-border-subtle">
      <div class="max-w-6xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-text-muted">
        <div class="flex items-center gap-3">
          <div class="w-6 h-6 border border-border-subtle rounded-xs flex items-center justify-center">
            <div class="w-2 h-2 rounded-xs bg-text-muted rotate-45" />
          </div>
          <span>digit-link</span>
        </div>
        <p>&copy; {{ new Date().getFullYear() }} digit-link. All rights reserved.</p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
	@reference "../../style.css";
.faq-enter-active,
.faq-leave-active {
  transition: all 0.2s ease-out;
  overflow: hidden;
}

.faq-enter-from,
.faq-leave-to {
  opacity: 0;
  max-height: 0;
  padding-bottom: 0;
}

.faq-enter-to,
.faq-leave-from {
  opacity: 1;
  max-height: 200px;
}

.btn-lg {
  @apply py-3 px-6 text-base;
}
</style>
