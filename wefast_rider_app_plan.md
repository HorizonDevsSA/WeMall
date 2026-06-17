# WeFast Rider Mobile Application: Enterprise-Grade Architecture & Implementation Plan

This document details the architecture, background services, APIs, and implementation plan for **WeFast Rider**, the dedicated application for crowdsourced gig-economy couriers and logistics drivers.

---

## 1. Why a Standalone Rider Application is Necessary

As a senior developer, combining the customer-facing delivery app (WeFast) and the rider app into a single APK/AAB is a critical anti-pattern for an enterprise-level platform. Here is the technical and operational rationale for keeping the WeFast Rider App separate:

### 1.1 Background Geolocation & Store Permission Scrutiny
*   **The Problem**: Rider apps must monitor coordinates in the background (even when the screen is locked or the app is closed) to fuel Redis Geo-matching and calculate ETAs. 
*   **The Scrutiny**: App stores (specifically Google Play console) enforce exhaustive reviews for apps requesting `ACCESS_BACKGROUND_LOCATION`. If a regular consumer app requests this permission, it will be rejected. By isolating driver code, we guarantee that only the Rider App requests background tracking permissions, making Store approval straightforward.

### 1.2 Resource Constraints and Binary Size (Bloatware Prevention)
*   **The Problem**: Rider applications require additional, heavy dependencies:
    - **OCR & Document Scanner Engines**: For driving license, vehicle plate, and ID registration.
    - **Telemetry & Gyroscope Analysis Libraries**: To monitor driver velocity, harsh braking, and vehicle speed.
    - **Offline Navigation Tiles**: Maps data for navigation in low-network spots.
*   **The Rationale**: Forcing these libraries onto regular consumers increases the APK download size by ~80MB, causing cart abandonment and low app retention.

### 1.3 Permissions and User Privacy Pushback
*   **The Rationale**: Consumers are highly sensitive to privacy. If a user delivery app requests permissions to start automatically on boot (`RECEIVE_BOOT_COMPLETED`), draw over other apps (for overlay dispatch panels), and bypass battery optimizations, users will uninstall the app out of privacy concerns.

### 1.4 Lifecycle and Release Autonomy
*   **The Rationale**: Rider apps have separate sprint backlogs and rapid hotfix requirements (e.g. patching Bluetooth thermal print drivers or resolving location drift on specific Xiaomi devices). Decoupling release cycles prevents courier bugs from delaying user-facing promotional deployments.

---

## 2. Architectural Blueprint (Clean Architecture & MVI)

The WeFast Rider application follows Clean Architecture, utilizing **Kotlin Coroutines** and **Jetpack Compose**. It shares common core data layers (such as auth tokens and graphql client bases) via shared core libraries.

```
                ┌──────────────────────────────────────────────┐
                │          PRESENTATION / UI LAYER             │
                │     (Compose UI, ViewModels, UI States)      │
                └──────────────────────┬───────────────────────┘
                                       │
                                       ▼ (Uses interfaces)
                ┌──────────────────────────────────────────────┐
                │                 DOMAIN LAYER                 │
                │   (Pure Kotlin: UseCases, Rider Contracts)   │
                └──────────────────────▲───────────────────────┘
                                       │
                                       │ (Implements interfaces)
                ┌──────────────────────┴───────────────────────┘
                │                  DATA LAYER                  │
                │ (Apollo GraphQL, SQLite Room, Location SDK)  │
                └──────────────────────────────────────────────┘
```

### 2.1 Directory Structure & Packages
```
wefast-rider/
├── core/
│   ├── designsystem/          # Dark-themed high-contrast driver UI
│   ├── network/               # Apollo client configured for WebSockets
│   ├── location/              # Foreground Service tracking lifecycle
│   └── database/              # Room database for storing offline route directions
└── features/
    ├── onboarding/            # Driver application forms, KYC submissions, vehicle details
    ├── dispatch/              # Bidding board, proximity job offers
    ├── fulfillment/           # Active job milestones, navigation maps, signature/photo captures
    └── wallet/                # Real-time earnings summary, cash-out routes
```

---

## 3. Key Driver Features & Technologies

| Feature Area | Technology | Implementation Details |
| :--- | :--- | :--- |
| **Real-time GPS Dispatch** | Google FusedLocationProviderClient | Requests GPS updates. Runs as a persistent **Foreground Service** using a system tray notification to prevent OS teardown. |
| **Job Proximity Search** | Redis Geohash + REST | Sends location coordinate ticks to the backend geo-index (`GEOADD active_couriers`) every 30 seconds. |
| **In-App Navigation** | Mapbox Navigator / Google Maps | Renders route directions directly in-app, overlays traffic polylines, and recalculates paths on off-route events. |
| **Signature / Proof** | Custom Canvas / CameraX | Captures recipient signatures on-screen or takes a photo of the package at the door for proof of delivery. |

---

## 4. GraphQL API Connections & Operations

The Rider App connects to WeMall's API Gateway. The key operations are defined below:

### 4.1 Courier Registration & Online Status Mutations
```graphql
mutation RegisterAsCourier($vehicleType: String!, $plateNumber: String) {
  registerAsCourier(vehicleType: $vehicleType, plateNumber: $plateNumber) {
    id
    vehicleType
    plateNumber
    isOnline
    rating
  }
}

mutation SetCourierOnlineStatus($isOnline: Boolean!) {
  setCourierOnlineStatus(isOnline: $isOnline) {
    id
    isOnline
  }
}
```

### 4.2 Proximity Job Retrieval
```graphql
query AvailableCourierTasks($latitude: Float!, $longitude: Float!) {
  availableCourierTasks(location: { latitude: $latitude, longitude: $longitude }) {
    id
    trackingNumber
    senderName
    senderAddressLine1
    recipientName
    recipientAddressLine1
    shippingFee
    weightKg
  }
}

mutation AcceptCourierTask($deliveryOrderId: ID!) {
  acceptCourierTask(deliveryOrderId: $deliveryOrderId)
}
```

### 4.3 Active Task Progression
```graphql
mutation UpdateDeliveryProgress(
  $deliveryOrderId: ID!
  $status: DeliveryStatus!
  $latitude: Float!
  $longitude: Float!
  $details: String
) {
  updateDeliveryProgress(
    deliveryOrderId: $deliveryOrderId
    status: $status
    location: { latitude: $latitude, longitude: $longitude }
    details: $details
  )
}
```

---

## 5. Critical Rider Loopholes & Enterprise Resolutions

### Loophole 1: GPS Spoofing & Cheating
*   **The Issue**: Couriers can download "Mock Location" apps to trick the system into thinking they are right next to high-value merchants, letting them monopolize short, lucrative deliveries.
*   **The Resolution**: The location tracking module uses `Location.isFromMockProvider()` on Android API 18-30, and `Location.isMock()` on Android 31+. The app runs a background security check: if mock locations are detected, the app blocks the driver from going online and flags their account for review.

### Loophole 2: OS App Suspension (Doze Mode & Battery Management)
*   **The Issue**: Aggressive OS battery saving managers (e.g. Huawei's EMUI, Samsung's One UI) put background apps to sleep. The rider app stops reporting locations, causing the routing engine to think the driver has gone offline.
*   **The Resolution**: The application must prompt the user to disable battery optimizations for the app using the `Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS` intent. The background service must also be declared as a `foregroundServiceType="location"` in `AndroidManifest.xml` and display a persistent status notification.

### Loophole 3: Network Dead-zones (Transit Connectivity Loss)
*   **The Issue**: Drivers lose network inside tunnels, remote neighborhoods, or metal-framed cargo elevators. If they deliver a package, they cannot register the "Delivered" status on the server.
*   **The Resolution**: Local SQLite tracking logs buffer changes offline. When the driver taps "Confirm Delivery" without network, the action is marked as `pending_sync` in the local Room DB along with a timestamp, signature/photo binary, and GPS coordinates. A `WorkManager` job polls network connectivity and pushes the buffered operations as soon as connection is restored.

---

## 6. Test-Driven Development (TDD) Blueprint

Below is the unit test targeting the driver verification and task state validation flow, ensuring a rider cannot transition a task to `DELIVERED` without providing a validation signature or photo.

```kotlin
class TaskFulfillmentUseCaseTest {

    private val repository: RiderRepository = mockk()
    private lateinit var completeTaskUseCase: CompleteTaskUseCase

    @BeforeEach
    fun setUp() {
        completeTaskUseCase = CompleteTaskUseCase(repository)
    }

    @Test
    fun `when completing delivery without signature or photo, returns validation exception`() = runTest {
        // Act
        val result = completeTaskUseCase(
            taskId = "t1",
            signatureBytes = null,
            photoUrl = null
        )

        // Assert
        assertTrue(result.isFailure)
        assertEquals("Proof of delivery required", result.exceptionOrNull()?.message)
        verify { repository wasNot Called }
    }

    @Test
    fun `when completing delivery with valid parameters, repository is updated`() = runTest {
        // Arrange
        val signature = byteArrayOf(1, 2, 3)
        coEvery { repository.completeTask("t1", signature, null) } returns Result.success(Unit)

        // Act
        val result = completeTaskUseCase("t1", signature, null)

        // Assert
        assertTrue(result.isSuccess)
        coVerify(exactly = 1) { repository.completeTask("t1", signature, null) }
    }
}
```
