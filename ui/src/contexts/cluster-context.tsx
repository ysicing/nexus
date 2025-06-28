import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'

interface Cluster {
  id: string
  name: string
  description?: string
  server: string
  version?: string
  status: 'healthy' | 'unhealthy' | 'unreachable' | 'unknown'
  isDefault: boolean
  labels?: Record<string, string>
  createdAt: string
  updatedAt: string
  lastCheck: string
  context?: string
}

interface ClusterStats {
  nodes: {
    total: number
    ready: number
  }
  pods: {
    total: number
    running: number
  }
  namespaces: {
    total: number
  }
}

interface ClusterContextType {
  clusters: Cluster[]
  selectedCluster: string | null
  currentCluster: Cluster | null
  isLoading: boolean
  error: Error | null
  setSelectedCluster: (clusterId: string) => void
  refreshClusters: () => void
  addCluster: (clusterData: {
    name: string
    description?: string
    kubeconfigContent: string
    labels?: Record<string, string>
  }) => Promise<Cluster>
  removeCluster: (clusterId: string) => Promise<void>
  setDefaultCluster: (clusterId: string) => Promise<void>
  updateClusterLabels: (clusterId: string, labels: Record<string, string>) => Promise<void>
  getClusterStats: (clusterId: string) => Promise<ClusterStats>
  useClusterAPI: () => {
    get: <T>(endpoint: string) => Promise<T>
    post: <T>(endpoint: string, data?: any) => Promise<T>
    put: <T>(endpoint: string, data?: any) => Promise<T>
    delete: <T>(endpoint: string) => Promise<T>
  }
}

const ClusterContext = createContext<ClusterContextType | undefined>(undefined)

const SELECTED_CLUSTER_KEY = 'nexus-selected-cluster'

export function ClusterProvider({ children }: { children: React.ReactNode }) {
  const [selectedCluster, setSelectedClusterState] = useState<string | null>(
    () => localStorage.getItem(SELECTED_CLUSTER_KEY)
  )
  const queryClient = useQueryClient()

  // 获取集群列表
  const {
    data: clustersData,
    isLoading,
    error,
    refetch: refreshClusters,
  } = useQuery({
    queryKey: ['clusters'],
    queryFn: async () => {
      const response = await apiClient.get<{
        clusters: Cluster[]
        total: number
      }>('/clusters')
      return response
    },
    staleTime: 30000, // 30秒内数据有效
    refetchInterval: 60000, // 每分钟自动刷新
  })

  const clusters = clustersData?.clusters || []
  const currentCluster = clusters.find(c => c.id === selectedCluster) || null

  // 自动选择默认集群
  useEffect(() => {
    if (clusters.length > 0 && !selectedCluster) {
      const defaultCluster = clusters.find(c => c.isDefault) || clusters[0]
      setSelectedCluster(defaultCluster.id)
    }
  }, [clusters, selectedCluster])

  const setSelectedCluster = useCallback((clusterId: string) => {
    setSelectedClusterState(clusterId)
    localStorage.setItem(SELECTED_CLUSTER_KEY, clusterId)
  }, [])

  const addCluster = useCallback(async (clusterData: {
    name: string
    description?: string
    kubeconfigContent: string
    labels?: Record<string, string>
  }): Promise<Cluster> => {
    const response = await apiClient.post<Cluster>('/clusters', clusterData)
    queryClient.invalidateQueries({ queryKey: ['clusters'] })
    return response
  }, [queryClient])

  const removeCluster = useCallback(async (clusterId: string): Promise<void> => {
    await apiClient.delete(`/clusters/${clusterId}`)
    queryClient.invalidateQueries({ queryKey: ['clusters'] })
    
    // 如果删除的是当前选中的集群，切换到其他集群
    if (selectedCluster === clusterId) {
      const remainingClusters = clusters.filter(c => c.id !== clusterId)
      if (remainingClusters.length > 0) {
        const defaultCluster = remainingClusters.find(c => c.isDefault) || remainingClusters[0]
        setSelectedCluster(defaultCluster.id)
      } else {
        setSelectedClusterState(null)
        localStorage.removeItem(SELECTED_CLUSTER_KEY)
      }
    }
  }, [queryClient, selectedCluster, clusters, setSelectedCluster])

  const setDefaultCluster = useCallback(async (clusterId: string): Promise<void> => {
    await apiClient.put(`/clusters/${clusterId}/default`)
    queryClient.invalidateQueries({ queryKey: ['clusters'] })
  }, [queryClient])

  const updateClusterLabels = useCallback(async (
    clusterId: string, 
    labels: Record<string, string>
  ): Promise<void> => {
    await apiClient.put(`/clusters/${clusterId}/labels`, { labels })
    queryClient.invalidateQueries({ queryKey: ['clusters'] })
  }, [queryClient])

  const getClusterStats = useCallback(async (clusterId: string): Promise<ClusterStats> => {
    return await apiClient.get<ClusterStats>(`/clusters/${clusterId}/stats`)
  }, [])

  // 提供带集群参数的 API 调用方法
  const useClusterAPI = useCallback(() => {
    const clusterParam = selectedCluster ? `?cluster=${selectedCluster}` : ''
    const clusterHeader = selectedCluster ? { 'X-Cluster-ID': selectedCluster } : undefined

    const api = {
      get: async function<T>(endpoint: string): Promise<T> {
        const url = endpoint.includes('?') 
          ? `${endpoint}&cluster=${selectedCluster || ''}` 
          : `${endpoint}${clusterParam}`
        return apiClient.get<T>(url, { headers: clusterHeader })
      },
      post: async function<T>(endpoint: string, data?: any): Promise<T> {
        const url = endpoint.includes('?') 
          ? `${endpoint}&cluster=${selectedCluster || ''}` 
          : `${endpoint}${clusterParam}`
        return apiClient.post<T>(url, data, { headers: clusterHeader })
      },
      put: async function<T>(endpoint: string, data?: any): Promise<T> {
        const url = endpoint.includes('?') 
          ? `${endpoint}&cluster=${selectedCluster || ''}` 
          : `${endpoint}${clusterParam}`
        return apiClient.put<T>(url, data, { headers: clusterHeader })
      },
      delete: async function<T>(endpoint: string): Promise<T> {
        const url = endpoint.includes('?') 
          ? `${endpoint}&cluster=${selectedCluster || ''}` 
          : `${endpoint}${clusterParam}`
        return apiClient.delete<T>(url, { headers: clusterHeader })
      },
    }
    return api
  }, [selectedCluster])

  const value: ClusterContextType = {
    clusters,
    selectedCluster,
    currentCluster,
    isLoading,
    error: error as Error | null,
    setSelectedCluster,
    refreshClusters,
    addCluster,
    removeCluster,
    setDefaultCluster,
    updateClusterLabels,
    getClusterStats,
    useClusterAPI,
  }

  return (
    <ClusterContext.Provider value={value}>
      {children}
    </ClusterContext.Provider>
  )
}

export function useCluster() {
  const context = useContext(ClusterContext)
  if (context === undefined) {
    throw new Error('useCluster must be used within a ClusterProvider')
  }
  return context
} 
