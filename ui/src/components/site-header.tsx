import { useState } from 'react'
import githubIcon from '@/assets/github.svg'
import { Plus } from 'lucide-react'

import { useIsMobile } from '@/hooks/use-mobile'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'

import { ColorThemeToggle } from './color-theme-toggle'
import { CreateResourceDialog } from './create-resource-dialog'
import { DynamicBreadcrumb } from './dynamic-breadcrumb'
import { ModeToggle } from './mode-toggle'
import { Search } from './search'
import { UserMenu } from './user-menu'
import { ClusterSelector } from './cluster-selector'
import { ClusterManagementDialog } from './cluster-management-dialog'
import { AddClusterDialog } from './add-cluster-dialog'

export function SiteHeader() {
  const isMobile = useIsMobile()
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [clusterManagementOpen, setClusterManagementOpen] = useState(false)
  const [addClusterOpen, setAddClusterOpen] = useState(false)

  return (
    <>
      <header className="sticky top-0 z-50 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
        <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
          <SidebarTrigger className="-ml-1" />
          <Separator
            orientation="vertical"
            className="mx-2 data-[orientation=vertical]:h-4"
          />
          <DynamicBreadcrumb />
          <div className="ml-auto flex items-center gap-2">
            {!isMobile && (
              <>
                <ClusterSelector
                  onAddCluster={() => setAddClusterOpen(true)}
                  onManageClusters={() => setClusterManagementOpen(true)}
                />
                <Separator
                  orientation="vertical"
                  className="mx-2 data-[orientation=vertical]:h-4"
                />
              </>
            )}
            <Search />
            <Plus
              className="h-5 w-5 cursor-pointer text-muted-foreground hover:text-foreground"
              onClick={() => setCreateDialogOpen(true)}
              aria-label="Create new resource"
            />
            {!isMobile && (
              <>
                <Separator
                  orientation="vertical"
                  className="mx-2 data-[orientation=vertical]:h-4"
                />
                <Button
                  variant="ghost"
                  asChild
                  size="sm"
                  className="hidden sm:flex"
                >
                  <a
                    href="https://github.com/ysicing/nexus"
                    aria-label="GitHub"
                    target="_blank"
                    className="dark:text-foreground"
                  >
                    <img
                      src={githubIcon}
                      alt="GitHub"
                      className="h-5 w-5 dark:invert"
                    />
                  </a>
                </Button>
                <ColorThemeToggle />
                <ModeToggle />
                <UserMenu />
              </>
            )}
          </div>
        </div>
      </header>

      <CreateResourceDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />
      
      <ClusterManagementDialog
        open={clusterManagementOpen}
        onOpenChange={setClusterManagementOpen}
      />
      
      <AddClusterDialog
        open={addClusterOpen}
        onOpenChange={setAddClusterOpen}
      />
    </>
  )
}
