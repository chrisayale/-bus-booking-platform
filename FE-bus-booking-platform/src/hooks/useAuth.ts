import { useSelector } from "react-redux"
import type { RootState } from "../store"

export function useAuth() {
  const { user, isAuthenticated, isLoading } = useSelector((state: RootState) => state.auth)

  return {
    user,
    isAuthenticated,
    isLoading,
    isOwner: isAuthenticated && user?.role === "owner",
    isPassenger: isAuthenticated && user?.role === "passenger",
    isPlatformAdmin: isAuthenticated && (user?.role === "platform_admin" || user?.role === "super_admin"),
    isSuperAdmin: isAuthenticated && user?.role === "super_admin",
  }
}