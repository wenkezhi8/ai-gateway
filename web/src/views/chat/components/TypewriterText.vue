<template>
  <div class="typewriter-text">
    <div class="content" v-html="renderedContent"></div>
    <span v-if="showCursor" class="cursor">|</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{
  content: string
  showCursor?: boolean
}>()

// Initialize markdown parser
const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: true
})

const renderedContent = computed(() => {
  if (!props.content) return ''
  return md.render(props.content)
})
</script>

<style lang="scss" scoped>
.typewriter-text {
  display: inline;
  line-height: 1.6;

  .content {
    display: inline;

    :deep(p) {
      margin: 0 0 0.5em 0;

      &:last-child {
        margin-bottom: 0;
      }
    }

    :deep(code) {
      background: var(--bg-tertiary);
      padding: 2px 6px;
      border-radius: var(--border-radius-sm);
      font-family: 'SF Mono', Monaco, 'Courier New', monospace;
      font-size: 0.9em;
    }

    :deep(pre) {
      background: var(--bg-tertiary);
      padding: var(--spacing-md);
      border-radius: var(--border-radius-md);
      overflow-x: auto;
      margin: var(--spacing-sm) 0;

      code {
        background: transparent;
        padding: 0;
      }
    }

    :deep(ul),
    :deep(ol) {
      margin: 0.5em 0;
      padding-left: 1.5em;
    }

    :deep(li) {
      margin: 0.25em 0;
    }

    :deep(a) {
      color: var(--color-primary);
      text-decoration: none;

      &:hover {
        text-decoration: underline;
      }
    }

    :deep(blockquote) {
      border-left: 3px solid var(--color-primary);
      padding-left: var(--spacing-md);
      margin: var(--spacing-sm) 0;
      color: var(--text-secondary);
    }

    :deep(table) {
      border-collapse: collapse;
      margin: var(--spacing-sm) 0;

      th,
      td {
        border: 1px solid var(--border-color);
        padding: var(--spacing-xs) var(--spacing-sm);
      }

      th {
        background: var(--bg-tertiary);
      }
    }
  }

  .cursor {
    display: inline-block;
    animation: blink 1s infinite;
    font-weight: 300;
    color: var(--color-primary);
    margin-left: 1px;
  }
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}
</style>
