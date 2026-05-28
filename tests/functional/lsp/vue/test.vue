<template>
  <div class="app">
    <p>{{ message }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const message = ref<string>('Hello')
const count: number = "invalid"
</script>
