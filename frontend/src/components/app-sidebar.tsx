import { Link, useRouterState } from "@tanstack/react-router"
import {
  BroadcastIcon,
  KeyIcon,
  LayoutIcon,
  TerminalWindowIcon,
} from "@phosphor-icons/react"

import { NavUser } from "@/components/nav-user"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { useDashboard } from "@/lib/dashboard-context"

const navItems = [
  {
    title: "Overview",
    to: "/dashboard" as const,
    icon: LayoutIcon,
  },
  {
    title: "Tunnels",
    to: "/dashboard/tunnels" as const,
    icon: BroadcastIcon,
    showOnlineCount: true,
  },
  {
    title: "CLI setup",
    to: "/dashboard/cli" as const,
    icon: TerminalWindowIcon,
  },
  {
    title: "Access",
    to: "/dashboard/access" as const,
    icon: KeyIcon,
  },
]

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { user, onlineCount, signOut } = useDashboard()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const name = user.email.split("@")[0] || user.email

  return (
    <Sidebar variant="inset" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              render={<Link to="/dashboard" />}
            >
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <BroadcastIcon className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">OpenTunnel</span>
                <span className="truncate text-xs">Workspace</span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Workspace</SidebarGroupLabel>
          <SidebarMenu>
            {navItems.map((item) => {
              const active =
                item.to === "/dashboard"
                  ? pathname === "/dashboard"
                  : pathname.startsWith(item.to)
              return (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton
                    isActive={active}
                    render={<Link to={item.to} />}
                    tooltip={item.title}
                  >
                    <item.icon />
                    <span>{item.title}</span>
                  </SidebarMenuButton>
                  {item.showOnlineCount && onlineCount ? (
                    <SidebarMenuBadge>{onlineCount}</SidebarMenuBadge>
                  ) : null}
                </SidebarMenuItem>
              )
            })}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser
          onSignOut={signOut}
          user={{
            name,
            email: user.email,
          }}
        />
      </SidebarFooter>
    </Sidebar>
  )
}
