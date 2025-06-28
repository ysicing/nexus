import { useState } from 'react'
import { Check, ChevronsUpDown, Plus, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Badge } from '@/components/ui/badge'
import { useCluster } from '@/contexts/cluster-context'

interface ClusterSelectorProps {
  selectedCluster?: string
  onClusterChange?: (clusterId: string) => void
  showAddButton?: boolean
  onAddCluster?: () => void
  onManageClusters?: () => void
}

export function ClusterSelector({
  selectedCluster: externalSelectedCluster,
  onClusterChange: externalOnClusterChange,
  showAddButton = true,
  onAddCluster,
  onManageClusters,
}: ClusterSelectorProps) {
  const [open, setOpen] = useState(false)
  const { 
    selectedCluster, 
    setSelectedCluster, 
    clusters, 
    isLoading, 
    currentCluster 
  } = useCluster()

  // 如果有外部传入的选中集群，使用外部的控制
  const actualSelectedCluster = externalSelectedCluster || selectedCluster
  const actualOnClusterChange = externalOnClusterChange || setSelectedCluster

  const handleClusterSelect = (clusterId: string) => {
    actualOnClusterChange(clusterId)
    setOpen(false)
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'bg-green-500'
      case 'unhealthy':
        return 'bg-yellow-500'
      case 'unreachable':
        return 'bg-red-500'
      default:
        return 'bg-gray-500'
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

  const selectedClusterData = clusters.find(c => c.id === actualSelectedCluster) || currentCluster

  if (isLoading) {
    return <div className="h-10 w-48 bg-gray-100 animate-pulse rounded-md" />
  }

  return (
    <div className="flex items-center gap-2">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className="w-64 justify-between"
          >
            {selectedClusterData ? (
              <div className="flex items-center gap-2 flex-1 min-w-0">
                <div
                  className={cn(
                    'w-2 h-2 rounded-full flex-shrink-0',
                    getStatusColor(selectedClusterData.status)
                  )}
                />
                <span className="truncate">{selectedClusterData.name}</span>
                {selectedClusterData.isDefault && (
                  <Badge variant="secondary" className="text-xs">
                    默认
                  </Badge>
                )}
              </div>
            ) : (
              <span className="text-muted-foreground">选择集群...</span>
            )}
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-80 p-0">
          <Command>
            <CommandInput placeholder="搜索集群..." />
            <CommandEmpty>未找到集群。</CommandEmpty>
            <CommandGroup>
              {clusters.map((cluster) => (
                <CommandItem
                  key={cluster.id}
                  value={cluster.id}
                  onSelect={() => handleClusterSelect(cluster.id)}
                  className="flex items-center gap-2 p-3"
                >
                                     <Check
                     className={cn(
                       'mr-2 h-4 w-4',
                       actualSelectedCluster === cluster.id ? 'opacity-100' : 'opacity-0'
                     )}
                   />
                  <div
                    className={cn(
                      'w-2 h-2 rounded-full flex-shrink-0',
                      getStatusColor(cluster.status)
                    )}
                  />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium truncate">{cluster.name}</span>
                      {cluster.isDefault && (
                        <Badge variant="secondary" className="text-xs">
                          默认
                        </Badge>
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      <div className="truncate">{cluster.server}</div>
                      <div className="flex items-center gap-2 mt-1">
                        <span>状态: {getStatusText(cluster.status)}</span>
                        {cluster.version && (
                          <span>版本: {cluster.version}</span>
                        )}
                      </div>
                    </div>
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          </Command>
          
          {(showAddButton || onManageClusters) && (
            <div className="border-t p-2 flex gap-2">
              {showAddButton && onAddCluster && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setOpen(false)
                    onAddCluster()
                  }}
                  className="flex-1"
                >
                  <Plus className="h-4 w-4 mr-2" />
                  添加集群
                </Button>
              )}
              {onManageClusters && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setOpen(false)
                    onManageClusters()
                  }}
                  className="flex-1"
                >
                  <Settings className="h-4 w-4 mr-2" />
                  管理集群
                </Button>
              )}
            </div>
          )}
        </PopoverContent>
      </Popover>
      
             {selectedClusterData && (
         <div className="text-sm text-muted-foreground">
           {clusters.length} 个集群
         </div>
       )}
    </div>
  )
} 
