
import { Activity, AlertCircle, Wifi, WifiOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface ClusterStatusIndicatorProps {
  status: 'healthy' | 'unhealthy' | 'unreachable' | 'unknown'
  cluster?: {
    name: string
    version?: string
    lastCheck: string
    server?: string
  }
  size?: 'sm' | 'md' | 'lg'
  showText?: boolean
  showTooltip?: boolean
}

export function ClusterStatusIndicator({
  status,
  cluster,
  size = 'md',
  showText = false,
  showTooltip = true,
}: ClusterStatusIndicatorProps) {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'healthy':
        return {
          color: 'bg-green-500',
          textColor: 'text-green-600',
          bgColor: 'bg-green-50',
          icon: Activity,
          text: '健康',
          description: '集群运行正常，所有节点可用',
        }
      case 'unhealthy':
        return {
          color: 'bg-yellow-500',
          textColor: 'text-yellow-600',
          bgColor: 'bg-yellow-50',
          icon: AlertCircle,
          text: '异常',
          description: '集群部分功能异常，部分节点不可用',
        }
      case 'unreachable':
        return {
          color: 'bg-red-500',
          textColor: 'text-red-600',
          bgColor: 'bg-red-50',
          icon: WifiOff,
          text: '不可达',
          description: '无法连接到集群，请检查网络和认证配置',
        }
      default:
        return {
          color: 'bg-gray-500',
          textColor: 'text-gray-600',
          bgColor: 'bg-gray-50',
          icon: Wifi,
          text: '未知',
          description: '集群状态未知，正在检查中',
        }
    }
  }

  const getSizeConfig = (size: string) => {
    switch (size) {
      case 'sm':
        return {
          dot: 'w-2 h-2',
          icon: 'h-3 w-3',
          text: 'text-xs',
          badge: 'text-xs px-2 py-1',
        }
      case 'lg':
        return {
          dot: 'w-4 h-4',
          icon: 'h-5 w-5',
          text: 'text-base',
          badge: 'text-sm px-3 py-1.5',
        }
      default:
        return {
          dot: 'w-3 h-3',
          icon: 'h-4 w-4',
          text: 'text-sm',
          badge: 'text-sm px-2.5 py-1',
        }
    }
  }

  const statusConfig = getStatusConfig(status)
  const sizeConfig = getSizeConfig(size)
  const Icon = statusConfig.icon

  const formatLastCheck = (lastCheck: string) => {
    const date = new Date(lastCheck)
    const now = new Date()
    const diffInMinutes = Math.floor((now.getTime() - date.getTime()) / (1000 * 60))
    
    if (diffInMinutes < 1) {
      return '刚刚'
    } else if (diffInMinutes < 60) {
      return `${diffInMinutes} 分钟前`
    } else if (diffInMinutes < 1440) {
      const hours = Math.floor(diffInMinutes / 60)
      return `${hours} 小时前`
    } else {
      return date.toLocaleString('zh-CN')
    }
  }

  const indicator = (
    <div className="flex items-center gap-2">
      {showText ? (
        <Badge
          variant="secondary"
          className={cn(
            'flex items-center gap-1.5',
            statusConfig.textColor,
            statusConfig.bgColor,
            sizeConfig.badge
          )}
        >
          <Icon className={sizeConfig.icon} />
          <span>{statusConfig.text}</span>
        </Badge>
      ) : (
        <div className="relative">
          <div
            className={cn(
              'rounded-full flex-shrink-0',
              statusConfig.color,
              sizeConfig.dot
            )}
          />
          {status === 'healthy' && (
            <div
              className={cn(
                'absolute inset-0 rounded-full animate-ping',
                statusConfig.color,
                'opacity-75'
              )}
            />
          )}
        </div>
      )}
    </div>
  )

  if (!showTooltip || !cluster) {
    return indicator
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          {indicator}
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          <div className="space-y-2">
            <div className="font-semibold">{cluster.name}</div>
            <div className="space-y-1 text-sm">
              <div className="flex items-center gap-2">
                <Icon className="h-3 w-3" />
                <span className={statusConfig.textColor}>{statusConfig.text}</span>
              </div>
              <div className="text-muted-foreground">
                {statusConfig.description}
              </div>
              {cluster.version && (
                <div>
                  <span className="text-muted-foreground">版本:</span> {cluster.version}
                </div>
              )}
              {cluster.server && (
                <div>
                  <span className="text-muted-foreground">服务器:</span>{' '}
                  <span className="font-mono text-xs">{cluster.server}</span>
                </div>
              )}
              <div>
                <span className="text-muted-foreground">最后检查:</span>{' '}
                {formatLastCheck(cluster.lastCheck)}
              </div>
            </div>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

// 简化版本，只显示状态点
export function ClusterStatusDot({
  status,
  size = 'md',
  className,
}: {
  status: 'healthy' | 'unhealthy' | 'unreachable' | 'unknown'
  size?: 'sm' | 'md' | 'lg'
  className?: string
}) {
  const statusConfig = getStatusConfig(status)
  const sizeConfig = getSizeConfig(size)

  return (
    <div className={cn('relative', className)}>
      <div
        className={cn(
          'rounded-full flex-shrink-0',
          statusConfig.color,
          sizeConfig.dot
        )}
      />
      {status === 'healthy' && (
        <div
          className={cn(
            'absolute inset-0 rounded-full animate-ping',
            statusConfig.color,
            'opacity-75'
          )}
        />
      )}
    </div>
  )
}

function getStatusConfig(status: string) {
  switch (status) {
    case 'healthy':
      return { color: 'bg-green-500' }
    case 'unhealthy':
      return { color: 'bg-yellow-500' }
    case 'unreachable':
      return { color: 'bg-red-500' }
    default:
      return { color: 'bg-gray-500' }
  }
}

function getSizeConfig(size: string) {
  switch (size) {
    case 'sm':
      return { dot: 'w-2 h-2' }
    case 'lg':
      return { dot: 'w-4 h-4' }
    default:
      return { dot: 'w-3 h-3' }
  }
} 
