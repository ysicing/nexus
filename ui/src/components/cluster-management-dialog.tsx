import { useState } from 'react'
import { Plus, Trash2, Star, Activity, Clock, Server } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { toast } from 'sonner'
import { AddClusterDialog } from './add-cluster-dialog'
import { cn } from '@/lib/utils'

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
}

interface ClusterManagementDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ClusterManagementDialog({
  open,
  onOpenChange,
}: ClusterManagementDialogProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [clusterToDelete, setClusterToDelete] = useState<Cluster | null>(null)
  const queryClient = useQueryClient()

  const { data: clustersData, isLoading } = useQuery({
    queryKey: ['clusters'],
    queryFn: () => apiClient.get<{ clusters: Cluster[], total: number }>('/api/v1/clusters'),
    enabled: open,
  })

  const clusters: Cluster[] = clustersData?.clusters || []

  const deleteClusterMutation = useMutation({
    mutationFn: (clusterId: string) =>
      apiClient.delete(`/api/v1/clusters/${clusterId}`),
    onSuccess: () => {
      toast.success('集群删除成功')
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
      setDeleteDialogOpen(false)
      setClusterToDelete(null)
    },
    onError: (error: any) => {
      toast.error(`删除集群失败: ${error.message}`)
    },
  })

  const setDefaultClusterMutation = useMutation({
    mutationFn: (clusterId: string) =>
      apiClient.put(`/api/v1/clusters/${clusterId}/default`),
    onSuccess: () => {
      toast.success('默认集群设置成功')
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
    },
    onError: (error: any) => {
      toast.error(`设置默认集群失败: ${error.message}`)
    },
  })

  const handleDeleteCluster = (cluster: Cluster) => {
    setClusterToDelete(cluster)
    setDeleteDialogOpen(true)
  }

  const handleSetDefaultCluster = (clusterId: string) => {
    setDefaultClusterMutation.mutate(clusterId)
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'text-green-600 bg-green-50'
      case 'unhealthy':
        return 'text-yellow-600 bg-yellow-50'
      case 'unreachable':
        return 'text-red-600 bg-red-50'
      default:
        return 'text-gray-600 bg-gray-50'
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'healthy':
        return '健康'
      case 'unhealthy':
        return '异常'
      case 'unreachable':
        return '不可达'
      default:
        return '未知'
    }
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN')
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-6xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>集群管理</DialogTitle>
            <DialogDescription>
              管理您的 Kubernetes 集群连接，添加新集群或修改现有集群配置。
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <div className="text-sm text-muted-foreground">
                共 {clusters.length} 个集群
              </div>
              <Button onClick={() => setAddDialogOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                添加集群
              </Button>
            </div>

            {isLoading ? (
              <div className="space-y-3">
                {[...Array(3)].map((_, i) => (
                  <div key={i} className="h-16 bg-gray-100 animate-pulse rounded-md" />
                ))}
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>集群名称</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>服务器</TableHead>
                    <TableHead>版本</TableHead>
                    <TableHead>最后检查</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {clusters.map((cluster) => (
                    <TableRow key={cluster.id}>
                      <TableCell>
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{cluster.name}</span>
                            {cluster.isDefault && (
                              <Badge variant="secondary" className="text-xs">
                                默认
                              </Badge>
                            )}
                          </div>
                          {cluster.description && (
                            <div className="text-sm text-muted-foreground">
                              {cluster.description}
                            </div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant="secondary"
                          className={cn('flex items-center gap-1 w-fit', getStatusColor(cluster.status))}
                        >
                          <Activity className="h-3 w-3" />
                          {getStatusText(cluster.status)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Server className="h-4 w-4 text-muted-foreground" />
                          <span className="font-mono text-sm">{cluster.server}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-sm">
                          {cluster.version || '-'}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                          <Clock className="h-4 w-4" />
                          {formatDate(cluster.lastCheck)}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          {!cluster.isDefault && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleSetDefaultCluster(cluster.id)}
                              disabled={setDefaultClusterMutation.isPending}
                            >
                              <Star className="h-4 w-4" />
                            </Button>
                          )}
                          {cluster.id !== 'in-cluster' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDeleteCluster(cluster)}
                              disabled={deleteClusterMutation.isPending}
                              className="text-red-600 hover:text-red-700"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}

            {clusters.length === 0 && !isLoading && (
              <div className="text-center py-8 text-muted-foreground">
                <Server className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>还没有配置任何集群</p>
                <p className="text-sm">点击"添加集群"开始添加您的第一个集群</p>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      <AddClusterDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        onSuccess={() => {
          queryClient.invalidateQueries({ queryKey: ['clusters'] })
        }}
      />

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除集群</AlertDialogTitle>
            <AlertDialogDescription>
              您确定要删除集群 "{clusterToDelete?.name}" 吗？
              此操作无法撤销，将移除该集群的所有配置信息。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (clusterToDelete) {
                  deleteClusterMutation.mutate(clusterToDelete.id)
                }
              }}
              className="bg-red-600 hover:bg-red-700"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
} 
