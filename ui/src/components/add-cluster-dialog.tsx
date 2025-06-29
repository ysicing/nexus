import React, { useState } from 'react'
import { Upload, FileText, AlertCircle } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Alert,
  AlertDescription,
} from '@/components/ui/alert'
import { useMutation } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'
import { toast } from 'sonner'

interface AddClusterDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function AddClusterDialog({
  open,
  onOpenChange,
  onSuccess,
}: AddClusterDialogProps) {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    kubeconfigContent: '',
  })
  const [dragOver, setDragOver] = useState(false)

  const addClusterMutation = useMutation({
    mutationFn: (data: typeof formData) =>
      apiClient.post('/clusters', data),
    onSuccess: () => {
      toast.success('集群添加成功')
      onSuccess?.()
      onOpenChange(false)
      resetForm()
    },
    onError: (error: any) => {
      toast.error(`添加集群失败: ${error.message}`)
    },
  })

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      kubeconfigContent: '',
    })
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.name.trim() || !formData.kubeconfigContent.trim()) {
      toast.error('请填写必填字段')
      return
    }
    addClusterMutation.mutate(formData)
  }

  const handleFileUpload = (file: File) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      const content = e.target?.result as string
      setFormData(prev => ({ ...prev, kubeconfigContent: content }))
    }
    reader.readAsText(file)
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    
    const files = Array.from(e.dataTransfer.files)
    const file = files[0]
    
    if (file && file.type === 'text/plain' || file.name.endsWith('.yaml') || file.name.endsWith('.yml')) {
      handleFileUpload(file)
    } else {
      toast.error('请上传有效的 kubeconfig 文件')
    }
  }

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      handleFileUpload(file)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>添加集群</DialogTitle>
          <DialogDescription>
            添加一个新的 Kubernetes 集群到您的管理列表中。
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">集群名称 *</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                placeholder="例如：生产环境集群"
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">描述</Label>
              <Input
                id="description"
                value={formData.description}
                onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                placeholder="集群的简短描述（可选）"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="kubeconfig">Kubeconfig 配置 *</Label>
              
              {/* 文件上传区域 */}
              <div
                className={`border-2 border-dashed rounded-lg p-6 transition-colors ${
                  dragOver
                    ? 'border-primary bg-primary/5'
                    : 'border-gray-300 hover:border-gray-400'
                }`}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
              >
                <div className="text-center">
                  <Upload className="mx-auto h-12 w-12 text-gray-400" />
                  <div className="mt-4">
                    <Label htmlFor="file-upload" className="cursor-pointer">
                      <span className="text-primary hover:text-primary/80">
                        点击上传文件
                      </span>
                      <span className="text-gray-500"> 或拖拽文件到此处</span>
                    </Label>
                    <input
                      id="file-upload"
                      type="file"
                      className="hidden"
                      accept=".yaml,.yml,.txt"
                      onChange={handleFileInputChange}
                    />
                  </div>
                  <p className="text-xs text-gray-500 mt-2">
                    支持 .yaml, .yml, .txt 格式的 kubeconfig 文件
                  </p>
                </div>
              </div>

              {/* 文本输入区域 */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <FileText className="h-4 w-4" />
                  <span className="text-sm font-medium">或直接粘贴配置内容：</span>
                </div>
                <Textarea
                  id="kubeconfig"
                  value={formData.kubeconfigContent}
                  onChange={(e) => setFormData(prev => ({ ...prev, kubeconfigContent: e.target.value }))}
                  placeholder="粘贴您的 kubeconfig 文件内容..."
                  className="min-h-32 font-mono text-sm"
                  required
                />
              </div>
            </div>
          </div>

          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              请确保您的 kubeconfig 文件包含有效的集群连接信息。
              添加集群后，系统会自动验证连接并检查集群状态。
            </AlertDescription>
          </Alert>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={addClusterMutation.isPending}
            >
              取消
            </Button>
            <Button
              type="submit"
              disabled={addClusterMutation.isPending || !formData.name.trim() || !formData.kubeconfigContent.trim()}
            >
              {addClusterMutation.isPending ? '添加中...' : '添加集群'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
} 
