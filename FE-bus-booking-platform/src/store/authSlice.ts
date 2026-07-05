import { createSlice, createAsyncThunk } from "@reduxjs/toolkit"
import type { PayloadAction } from "@reduxjs/toolkit"
import { api, setTokens, clearTokens } from "../api/client"
import type { User, AuthResponse } from "../types"

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
}

const initialState: AuthState = {
  user: null,
  isAuthenticated: !!localStorage.getItem("access_token"),
  isLoading: !!localStorage.getItem("access_token"), // loading until profile check completes
  error: null,
}

export const login = createAsyncThunk(
  "auth/login",
  async (credentials: { email: string; password: string }, { rejectWithValue }) => {
    try {
      const data = await api.post<AuthResponse>("/auth/login", credentials)
      setTokens(data.access_token, data.refresh_token)
      return data.user
    } catch (err: unknown) {
      return rejectWithValue((err as Error).message)
    }
  }
)

export const register = createAsyncThunk(
  "auth/register",
  async (
    credentials: { email: string; password: string; full_name: string; phone?: string },
    { rejectWithValue }
  ) => {
    try {
      await api.post("/auth/register", credentials)
      return true
    } catch (err: unknown) {
      return rejectWithValue((err as Error).message)
    }
  }
)

// Fetch user profile on page refresh to restore user state (including role)
export const fetchProfile = createAsyncThunk(
  "auth/fetchProfile",
  async (_, { rejectWithValue }) => {
    try {
      const user = await api.get<User>("/auth/profile")
      return user
    } catch (err: unknown) {
      clearTokens()
      return rejectWithValue((err as Error).message)
    }
  }
)

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    logout(state) {
      state.user = null
      state.isAuthenticated = false
      state.isLoading = false
      clearTokens()
    },
    setUser(state, action: PayloadAction<User>) {
      state.user = action.payload
      state.isAuthenticated = true
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(login.pending, (state) => {
        state.isLoading = true
        state.error = null
      })
      .addCase(login.fulfilled, (state, action) => {
        state.isLoading = false
        state.user = action.payload
        state.isAuthenticated = true
      })
      .addCase(login.rejected, (state, action) => {
        state.isLoading = false
        state.error = action.payload as string
      })
      .addCase(register.pending, (state) => {
        state.isLoading = true
        state.error = null
      })
      .addCase(register.fulfilled, (state) => {
        state.isLoading = false
      })
      .addCase(register.rejected, (state, action) => {
        state.isLoading = false
        state.error = action.payload as string
      })
      // Fetch profile (on page refresh)
      .addCase(fetchProfile.pending, (state) => {
        state.isLoading = true
      })
      .addCase(fetchProfile.fulfilled, (state, action) => {
        state.isLoading = false
        state.user = action.payload
        state.isAuthenticated = true
      })
      .addCase(fetchProfile.rejected, (state) => {
        state.isLoading = false
        state.user = null
        state.isAuthenticated = false
        // Tokens are already cleared in the thunk
      })
  },
})

export const { logout, setUser } = authSlice.actions
export default authSlice.reducer
