# WeFast Station Portal Application: Enterprise-Grade Architecture & Implementation Plan

This document details the architecture, UI layout rules, hardware integrations, APIs, and implementation plan for the **WeFast Station Portal** application. 

---

## 1. Why a Standalone Station Application is Necessary

The Station App is operated by **Post Station Keepers** (convenience stores, small shops, local pharmacies) who manage physical package inventory. It should **never** be combined with the consumer app for the following reasons:

### 1.1 Form Factor and Hardware Optimization
*   **The Problem**: Unlike consumers and couriers who use mobile phones, Station Keepers operate at a fixed desk/counter. They use **Android Tablets**, desktop computers, or specialized **POS (Point of Sale) terminals**. 
*   **The Layout**: The application must utilize a **tablet-first UI (Dual-Pane/Grid)**. Combining it with a phone-centric consumer app leads to poorly optimized screens, wasting screen real estate.
*   **Hardware Hooks**: Station apps require integration with external hardware:
    - **Physical Bluetooth/USB Barcode Scanners** (HID Mode).
    - **Thermal Waybill Printers** (ESC/POS print commands).
    - **External USB Weight Scales**.
    Consumer apps do not need these low-level device communication drivers.

### 1.2 Access Control and Data Security
*   **The Problem**: Keepers have access to bulk client data, package routing zones, shelf inventory indices, and telephone numbers.
*   **The Rationale**: Storing station data access privileges inside the consumer app bundle presents a massive security risk. A standalone app ensures that only verified Keepers with correct organizational credentials can authenticate and access the API Gateway's station nodes.

### 1.3 Operation Lifecycle
*   **The Rationale**: Station apps run continuously on mounted terminals throughout the store's opening hours (e.g., 14 hours a day). The app requires optimization for permanent power connection (preventing screen burn-in, locking screen rotation, and managing heavy processing loads during bulk morning sorting).

---

## 2. Architectural Blueprint (Clean Architecture & Tablet-MVI)

The app is built as an Android Tablet application using Clean Architecture, utilizing **Jetpack Compose Adaptive Layouts**.

```
                ┌──────────────────────────────────────────────┐
                │          PRESENTATION / UI LAYER             │
                │   (Compose Adaptive Dual-Pane UI, VM, MVI)   │
                └──────────────────────┬───────────────────────┘
                                       │
                                       ▼ (Uses interfaces)
                ┌──────────────────────────────────────────────┐
                │                 DOMAIN LAYER                 │
                │  (Pure Kotlin: UseCases, Inventory Rules)    │
                └──────────────────────▲───────────────────────┘
                                       │
                                       │ (Implements interfaces)
                ┌──────────────────────┴───────────────────────┘
                │                  DATA LAYER                  │
                │ (Apollo GraphQL, SQLite Room, Bluetooth SDK) │
                └──────────────────────────────────────────────┘
```

### 2.1 Directory Structure & Packages
```
wefast-station/
├── core/
│   ├── designsystem/          # Tablet components, high-density grids
│   ├── hardware/              # Bluetooth HID scanners & thermal printer adapters
│   └── network/               # Apollo client configured for secure Gateway routes
└── features/
    ├── checkin/               # Inbound package scanning, shelf routing assignments
    ├── checkout/              # Buyer OTP validation, outbound package handovers
    ├── inventory/             # Shelf visualizer, search & reconcile features
    └── analytics/             # Package volume charts, pickup latency metrics
```

---

## 3. Key Station Features & Technologies

| Feature Area | Technology | Implementation Details |
| :--- | :--- | :--- |
| **High-Speed Camera Scanning** | Google ML Kit Barcode Scanning | Configures an in-app CameraX analyzer. Detects Code 128 barcodes within 100ms and overlays virtual bounding boxes. |
| **Physical Scanner Support** | Android KeyEvents Interface | Overrides the Activity's `dispatchKeyEvent` to capture text inputs from USB/Bluetooth hardware scanners (which acts as a keyboard) and parses tracking numbers instantly. |
| **Label Thermal Printing** | ESC/POS Command Generator | Sends binary print buffers directly to Bluetooth/USB thermal printers (100mm x 150mm waybills). |
| **Adaptive Grid Visuals** | Jetpack Compose LazyVerticalGrid | Renders a visual map of the store's physical racks (e.g. Row-A, Shelf-3) showing capacity indicators. |

---

## 4. GraphQL API Connections & Operations

The Station Portal connects to WeMall's API Gateway. The key operations are defined below:

### 4.1 Package Check-In (Inbound)
```graphql
mutation StationCheckInPackage(
  $stationId: ID!
  $trackingNumber: String!
  $shelfCode: String!
  $direction: String!
) {
  stationCheckInPackage(
    stationId: $stationId
    trackingNumber: $trackingNumber
    shelfCode: $shelfCode
    direction: $direction
  ) {
    id
    shelfCode
    checkInAt
    deliveryOrder {
      trackingNumber
      recipientName
      status
    }
  }
}
```

### 4.2 Package Check-Out (Outbound)
```graphql
mutation StationCheckOutPackage(
  $stationId: ID!
  $trackingNumber: String!
  $verificationCode: String!
) {
  stationCheckOutPackage(
    stationId: $stationId
    trackingNumber: $trackingNumber
    verificationCode: $verificationCode
  )
}
```

### 4.3 Station Inventory Retrieval
```graphql
query StationInventory($stationId: ID!, $unclaimedOnly: Boolean!) {
  stationInventory(stationId: $stationId, unclaimedOnly: $unclaimedOnly) {
    id
    shelfCode
    checkInAt
    deliveryOrder {
      id
      trackingNumber
      recipientName
      recipientPhone
      status
    }
  }
}
```

---

## 5. Critical Station Loopholes & Enterprise Resolutions

### Loophole 1: External Keyboard Hardware Collision
*   **The Issue**: When a physical barcode scanner is paired via Bluetooth/USB in HID mode, the Android OS recognizes it as a physical keyboard. As a result, the OS automatically hides the virtual on-screen keyboard, preventing the Keeper from typing numbers manually when a waybill barcode is smudged.
*   **The Resolution**: Configure the system window to ignore physical keyboards when determining soft-keyboard visibility. In Compose, handle the barcode parsing globally by listening to text streams ending with `\n` (carriage return sent by scanners on completion) using a custom `KeyEvent` handler in the main Activity, bypassing visual text field focus.

### Loophole 2: Scanning Latency & Multi-Scan Concurrency
*   **The Issue**: During bulk sorting (e.g. 500 packages arriving from a truck), waiting for a network GraphQL request after each camera scan delays the operator.
*   **The Resolution**: Implement an asynchronous Scan Queue. The scanner adds waybills to a local Room table (`scanned_packages`) immediately with a UI sound check. A background worker picks up scanned entries and uploads them to the server in batches (`GraphQL Mutations`) or processes them concurrently with retry-backoff policies.

### Loophole 3: Package Shelving Discrepancies
*   **The Issue**: Packages are checked into `Row-A3` but accidentally placed on `Row-B1`. Finding them takes hours.
*   **The Resolution**: The app features a "Reconcile Shelf" audit mode. The operator scans a shelf's QR/barcode (e.g., `Row-A3`), then scans all packages sitting on that shelf. The app compares the scans against the server database and flags any discrepancies (e.g., "Package WM-108 is registered on Row-B1, please move it").

---

## 6. Test-Driven Development (TDD) Blueprint

Below is the unit test targeting the check-in business rules, ensuring that inbound packages are assigned a non-empty, formatted shelf coordinate.

```kotlin
class CheckInPackageUseCaseTest {

    private val repository: StationRepository = mockk()
    private lateinit var checkInPackageUseCase: CheckInPackageUseCase

    @BeforeEach
    fun setUp() {
        checkInPackageUseCase = CheckInPackageUseCase(repository)
    }

    @Test
    fun `when shelf code format is invalid, returns validation exception`() = runTest {
        // Act
        val result = checkInPackageUseCase(
            stationId = "s1",
            trackingNo = "WM-123",
            shelfCode = "", // Empty shelf code
            direction = "INBOUND"
        )

        // Assert
        assertTrue(result.isFailure)
        assertEquals("Shelf code cannot be empty", result.exceptionOrNull()?.message)
        verify { repository wasNot Called }
    }

    @Test
    fun `when shelf code is valid, repository completes package registration`() = runTest {
        // Arrange
        val expectedPackage = StationPackage(
            id = "pkg1",
            stationId = "s1",
            trackingNo = "WM-123",
            shelfCode = "Row-B2",
            checkInAt = "2026-06-16T20:00:00Z"
        )
        coEvery { 
            repository.checkIn("s1", "WM-123", "Row-B2", "INBOUND") 
        } returns Result.success(expectedPackage)

        // Act
        val result = checkInPackageUseCase("s1", "WM-123", "Row-B2", "INBOUND")

        // Assert
        assertTrue(result.isSuccess)
        assertEquals(expectedPackage, result.getOrNull())
        coVerify(exactly = 1) { repository.checkIn("s1", "WM-123", "Row-B2", "INBOUND") }
    }
}
```
