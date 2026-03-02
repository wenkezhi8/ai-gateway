import { defineStore } from 'pinia'
import type { VectorCollection } from '@/api/vector-db-domain'

interface VectorDbState {
  collections: VectorCollection[]
}

export const useVectorDbStore = defineStore('vector-db-domain', {
  state: (): VectorDbState => ({
    collections: []
  }),
  getters: {
    collectionCount: (state) => state.collections.length,
    totalVectors: (state) => state.collections.reduce((sum, item) => sum + (item.vector_count || 0), 0)
  },
  actions: {
    setCollections(items: VectorCollection[]) {
      this.collections = Array.isArray(items) ? items : []
    }
  }
})
