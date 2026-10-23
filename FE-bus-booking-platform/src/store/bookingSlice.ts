import { createSlice, createAsyncThunk } from "@reduxjs/toolkit"
import type { PayloadAction } from "@reduxjs/toolkit"
import { api } from "../api/client"
import type { Journey, Booking, SearchParams } from "../types"

interface BookingState {
  searchResults: Journey[]
  bookings: Booking[]
  selectedJourney: Journey | null
  isSearching: boolean
  isLoading: boolean
  error: string | null
}

const initialState: BookingState = {
  searchResults: [],
  bookings: [],
  selectedJourney: null,
  isSearching: false,
  isLoading: false,
  error: null,
}

export const searchTrips = createAsyncThunk(
  "booking/search",
  async (params: SearchParams, { rejectWithValue }) => {
    try {
      const query = new URLSearchParams({ origin: params.origin, destination: params.destination, date: params.date })
      return await api.get<Journey[]>(`/search/trips?${query}`)
    } catch (err: unknown) {
      return rejectWithValue((err as Error).message)
    }
  }
)

export const fetchBookings = createAsyncThunk(
  "booking/fetchBookings",
  async (_, { rejectWithValue }) => {
    try {
      return await api.get<Booking[]>("/bookings")
    } catch (err: unknown) {
      return rejectWithValue((err as Error).message)
    }
  }
)

const bookingSlice = createSlice({
  name: "booking",
  initialState,
  reducers: {
    setSelectedJourney(state, action: PayloadAction<Journey | null>) {
      state.selectedJourney = action.payload
    },
    clearSearch(state) {
      state.searchResults = []
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(searchTrips.pending, (state) => {
        state.isSearching = true
        state.error = null
      })
      .addCase(searchTrips.fulfilled, (state, action) => {
        state.isSearching = false
        state.searchResults = action.payload
      })
      .addCase(searchTrips.rejected, (state, action) => {
        state.isSearching = false
        state.error = action.payload as string
      })
      .addCase(fetchBookings.pending, (state) => {
        state.isLoading = true
      })
      .addCase(fetchBookings.fulfilled, (state, action) => {
        state.isLoading = false
        state.bookings = action.payload
      })
      .addCase(fetchBookings.rejected, (state, action) => {
        state.isLoading = false
        state.error = action.payload as string
      })
  },
})

export const { setSelectedJourney, clearSearch } = bookingSlice.actions
export default bookingSlice.reducer
