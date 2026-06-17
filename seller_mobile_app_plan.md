# WeMall Seller Mobile Application: Enterprise-Grade Architecture & Implementation Plan

This document outlines the architecture, code organization, backend integration, responsive design strategy, and testing workflows for the WeMall Seller mobile client. 

Built for business owners, this application prioritizes stability, high-performance catalog operations, offline functionality, real-time fulfillment alerts, and tablet-optimized layout adapters.

---

## 1. Architectural Blueprint (Clean Architecture & MVI)

The app is built around **Clean Architecture** principles, enforcing a strict unidirectional dependency rule: **UI/Presentation → Domain ← Data**.

```
                   ┌──────────────────────────────────────┐
                   │          PRESENTATION LAYER          │
                   │  (Compose, ViewModels, UI States)   │
                   └──────────────────┬───────────────────┘
                                      │
                                      ▼ (Uses interfaces)
                   ┌──────────────────────────────────────┐
                   │             DOMAIN LAYER             │
                   │  (Pure Kotlin: UseCases, Models)     │
                   └──────────────────▲───────────────────┘
                                      │
                                      │ (Implements interfaces)
                   ┌──────────────────┴───────────────────┘
                   │              DATA LAYER              │
                   │    (Apollo, Room, Repositories)      │
                   └──────────────────────────────────────┘
```

### Architectural Goals:
1. **Zero Framework Dependencies in Domain**: The Domain layer consists of pure Kotlin code. It defines business logic and repository interfaces without importing any Android frameworks (such as Context or ViewModels). This ensures absolute portability and ease of unit testing.
2. **Unidirectional Data Flow (MVI)**: User actions are captured as *Intents*. These intents are sent to the `ViewModel`, which processes them, updates the *UI State*, and exposes a single read-only stream (`StateFlow`) back to the Compose UI.
3. **Offline-First Capabilities**: Use cases read first from the local database (Room Cache). A background sync manager schedules data refreshes from the network and updates the local cache, ensuring the app remains responsive in offline/low-connectivity environments (such as logistics hubs).

---

## 2. Directory Structure & Codebase Organization

The codebase is organized using a **feature-by-feature** structure within the main Clean Architecture layers. This makes the project highly modular, preventing bloated packages and enabling developers to locate feature code (along with its domain logic, views, and repositories) instantly.

```
app/
├── src/
│   ├── main/
│   │   ├── java/com/wemall/seller/
│   │   │   ├── core/                      // Core/Shared cross-cutting modules
│   │   │   │   ├── designsystem/          // Typography, colors, theme, and common UI widgets (Buttons, Inputs)
│   │   │   │   ├── di/                    // Hilt modules (NetworkModule, DatabaseModule, RepositoryModule)
│   │   │   │   ├── network/               // Apollo GraphQL Client configuration, JWT interceptors, WebSocket engine
│   │   │   │   └── utils/                 // Extension utilities, dynamic spacing, window-size-class adapters
│   │   │   │
│   │   │   ├── data/                      // Data Layer (Implements Domain repositories)
│   │   │   │   ├── local/                 // Room DB tables, DAOs, and EncryptedSharedPreferences (credentials)
│   │   │   │   ├── remote/                // GraphQL operations (Apollo generated types)
│   │   │   │   ├── mapper/                // Data-to-Domain converters (maps DTOs directly to business models)
│   │   │   │   └── repository/            // Repository implementations (e.g., ProductRepositoryImpl)
│   │   │   │
│   │   │   ├── domain/                    // Domain Layer (Business Rules)
│   │   │   │   ├── model/                 // Immutable domain models (Store, Product, Order, User, Payout)
│   │   │   │   ├── repository/            // Repository Interfaces (contracts implemented by the Data layer)
│   │   │   │   └── usecase/               // Single-action business use cases (e.g., CreateProductUseCase, GetStoreDashboardUseCase)
│   │   │   │
│   │   │   └── presentation/              // Presentation / UI Layer (Compose & ViewModels)
│   │   │       ├── auth/                  // Onboarding, Register, Login
│   │   │       ├── dashboard/             // Sales charts, DSR metrics, payouts
│   │   │       ├── products/              // Product list grid, details view, multi-variant creation forms
│   │   │       ├── orders/                // Order list, fulfillment processing, delivery tracking
│   │   │       ├── chat/                  // Customer service threads, message bubble lists
│   │   │       └── navigation/            // NavHost, Route destinations, Adaptive navigation scaffolding
│   │   │
│   │   └── res/                           // Drawables, string resources, XML configurations
│   │
│   └── test/                              // Local Unit Tests (Run on JVM, TDD Target)
│   │   ├── domain/usecase/                // Test business logic (mocks Repositories)
│   │   ├── data/repository/               // Test API calls, database operations, error mapping
│   │   └── presentation/viewmodel/        // Test UI State transitions using Turbine (Flow inspection)
│   │
│   └── androidTest/                       // Instrumented UI & Integration Tests
│       ├── presentation/                  // Screen component Compose UI tests (ComposeTestRule)
│       └── local/                         // Room database CRUD integration tests
```

---

## 3. Technology Stack & Libraries

To build a premium, highly testable Android application, the following stack has been curated:

| Category | Technology | Rationale |
| :--- | :--- | :--- |
| **Language** | Kotlin | Standard language, modern syntax, native support for Coroutines/Flows. |
| **UI Toolkit** | Jetpack Compose (M3) | Declarative UI, reduces boilerplate, rich support for responsive layouts. |
| **DI Engine** | Hilt / Dagger | Standard dependency injection library for Android; simplifies VM and repository scoping. |
| **Network Client** | Apollo Kotlin (GraphQL) | Generates type-safe Kotlin models from GraphQL schemas, handles caching, and supports WebSocket subscriptions. |
| **Local Cache** | Room Database | SQLite wrapper, compiles schemas at build time, returns Flow streams for reactive UI. |
| **Security** | EncryptedSharedPreferences | Part of Android Jetpack Security; encrypts OAuth JWT tokens in the keystore. |
| **Image Loading** | Coil | Compose-first image loader; integrates natively with HTTP headers (for presigned URL requests). |
| **Testing Core** | JUnit 5 & MockK | Modern testing framework; MockK provides powerful mocking capabilities for Kotlin coroutines. |
| **Flow Testing** | Turbine | A small, elegant library for testing Kotlin Flow outputs without manually subscribing. |

---

## 4. Backend Alignment & Connection Strategy

The Seller App connects directly to the **WeMall API Gateway** (`http://localhost:8080/graphql`) using GraphQL. Below are the key endpoints we hook into, alongside proposed gateway extensions to support a complete seller workspace.

### A. Authentication & Onboarding
Sellers undergo credential authentication. The app captures JWT tokens, storing them securely in `EncryptedSharedPreferences` and appending them to all outgoing operations via an Apollo `HttpInterceptor`.
*   **Sign-Up / Sign-In Mutations**:
    - `sellerRegister(email, password, fullName)`
    - `sellerLogin(email, password)`
*   **Store Creation**: Upon first login, if the seller doesn't have an active storefront, they are redirected to a store setup wizard using `createStore(input: CreateStoreInput!)`.

### B. Catalog Management
Sellers manage products, variants (SKUs, price, size, color options), and initial stock levels.
*   **Listing Store Products**: `products(filter: ProductFilterInput { sellerId: $storeId })`
*   **Mutations**:
    - `createProduct(input: CreateProductInput!)` → creates products and nested variants.
    - `updateProduct(id, input: UpdateProductInput!)` → handles edits (modifying metadata or updating product status: `DRAFT`, `ACTIVE`, `PAUSED`).
    - `deleteProduct(id)` → soft-deletes a listing.
*   **Media Handling**: Up-to-date image assets are essential. The app uses the following flow to upload images directly:
    1. Send `requestUploadUrl(fileName: "product.jpg", mimeType: "image/jpeg")` to get a presigned S3 PUT URL.
    2. Perform a raw HTTP PUT request to upload the binary file.
    3. Call `confirmUpload(fileKey: "...")` to notify the media pipeline to generate CDN variants.

### C. Order Fulfillment & Payouts (API Alignment Suggestions)
While the current GraphQL gateway features extensive buyer-facing order queries, the seller app requires a dedicated workspace for merchant orders.
*   **Proposed Gateway Queries / Mutations**:
    - `sellerOrders(pageSize: Int, pageToken: String, status: OrderStatus): OrderList!` → Retrieves orders containing items supplied by the seller.
    - `updateOrderStatus(orderId: ID!, itemId: ID!, status: OrderStatus!): Order!` → Allows the seller to mark order items as `CONFIRMED` or `SHIPPED` (triggering logistics e-waybills).
*   **Payout Tracking (Active)**: `myStore { dsr, totalSales }` coupled with `listPayouts` and `getPayout` allows tracking settled earnings and historical bank transfers.

### D. Real-Time Chat & Customer Service
Communication between buyers and sellers is powered by WebSockets via Apollo Subscriptions. The UI subscribes to new messages instantly:
*   `myChatThreads` → Retrieves message boards grouped by buyer ID.
*   `chatMessages(threadId: ...)` → Pulls message history.
*   `sendChatMessage(threadId: ..., content: ...)` → Posts replies immediately.

---

## 5. Responsive Design Strategy (Mobile & Tablet)

A single codebase must dynamically adapt its presentation structure between phones and tablets to maximize screen real estate, especially when visualizing catalog grids or fulfillment dashboards.

```
                  ┌────────────────────────────────────────┐
                  │          WINDOW WIDTH CLASS            │
                  └──────────────────┬─────────────────────┘
                                     │
            ┌────────────────────────┼────────────────────────┐
            ▼ (<600dp)               ▼ (600dp - 839dp)        ▼ (>=840dp)
         COMPACT                  MEDIUM                  EXPANDED
     - Bottom Navigation      - Navigation Rail       - Permanent Drawer
     - Single-Pane Flow       - Single/Dual Pane      - Dual-Pane Layouts
```

### A. Navigation Adaptation
WeMall Seller uses **Window Width Size Classes** (`androidx.compose.material3.windowsizeclass.WindowWidthSizeClass`) to restructure navigation anchors:
*   **Compact (Phones)**: Bottom Navigation Bar. Provides thumbs-reach access to critical screens.
*   **Medium (Foldables, Small Tablets in Portrait)**: Vertical Navigation Rail on the left side of the screen.
*   **Expanded (Large Tablets in Landscape)**: Permanent Navigation Drawer. Offers a full, collapsible list of text-labeled options (Dashboard, Catalog, Orders, Chat, Settings) alongside the main workspace.

### B. Workspace Layout Adaptations
1. **Dual-Pane (List-Detail) Pattern**:
   *   *Features*: Order List + Order Fulfillment details, Chat Thread List + Active Conversation.
   - *On Compact*: Tapping an item navigates to a separate screen.
   - *On Expanded*: Displays a split screen (35% left pane for the list, 65% right pane for detail interaction). The app utilizes `androidx.compose.material3.adaptive:adaptive` layout wrappers to implement this cleanly.
2. **Dynamic Product Grids**:
   *   The product list feed adapts column counts to avoid stretched cards or empty voids.
   - *Compact*: 2 columns.
   - *Medium*: 3 columns.
   - *Expanded*: 5 columns.
3. **Adaptive Catalog Forms**:
   *   Creating or editing products requires complex inputs (images, attributes, variants, shipping metadata).
   - *Compact*: A single vertical-scrolling layout.
   - *Expanded*: Two equal-width horizontal columns. Left column contains image upload slots and variant matrices; right column houses text fields, categories, and tags.

### C. Compose Adaptive Navigation Blueprint

Here is the code structure implementing this responsive navigation wrapper:

```kotlin
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SellerAppContent(
    widthSizeClass: WindowWidthSizeClass,
    navController: NavHostController = rememberNavController()
) {
    val currentRoute = navController.currentBackStackEntryAsState().value?.destination?.route

    Row(modifier = Modifier.fillMaxSize()) {
        // Render Left Navigation Anchor for Medium & Expanded screens
        if (widthSizeClass == WindowWidthSizeClass.Medium) {
            SellerNavigationRail(
                currentRoute = currentRoute,
                onNavigate = { route -> navController.navigate(route) }
            )
        } else if (widthSizeClass == WindowWidthSizeClass.Expanded) {
            SellerNavigationDrawer(
                currentRoute = currentRoute,
                onNavigate = { route -> navController.navigate(route) }
            )
        }

        // Main Workspace Area
        Scaffold(
            bottomBar = {
                // Render Bottom Navigation only on Compact screens
                if (widthSizeClass == WindowWidthSizeClass.Compact) {
                    SellerBottomBar(
                        currentRoute = currentRoute,
                        onNavigate = { route -> navController.navigate(route) }
                    )
                }
            }
        ) { innerPadding ->
            SellerNavHost(
                navController = navController,
                widthSizeClass = widthSizeClass,
                modifier = Modifier.padding(innerPadding)
            )
        }
    }
}
```

---

## 6. Test-Driven Development (TDD) Workflow

To guarantee code quality, WeMall Seller mandates a **TDD-first** approach for all core logic. The workflow follows the strict **Red-Green-Refactor** pattern:

```
    1. Write Test (Fails) ──▶ 2. Implement Minimal Code (Passes) ──▶ 3. Refactor (Maintains Green)
           ▲                                                                    │
           └────────────────────────────────────────────────────────────────────┘
```

### A. Testing Strategy per Clean Architecture Layer

1.  **Domain Layer Use Cases (Unit Tests - Red/Green)**:
    *   *Approach*: Write use case tests first by mocking repository interfaces using `MockK`. Tests must verify success pathways, business validation rules, and error mappings.
    *   *Requirements*: 100% code coverage. Tests compile on the local JVM with zero Android dependencies.
2.  **Presentation ViewModels (MVI State Tests - Red/Green)**:
    *   *Approach*: Inject mock use cases into ViewModels. Fire *Intents* and use the **Turbine** library to collect the `uiState` flow, verifying that state transitions correspond exactly to the actions executed.
3.  **Data Layer Repositories (Integration/Unit Tests)**:
    *   *Approach*: Test mappers using standard unit tests. Test local caching behavior using in-memory Room database instances inside `androidTest` classes.

### B. Blueprint of a ViewModel MVI Test (Using Turbine & MockK)

Below is an enterprise-grade example testing a product deletion action, demonstrating the TDD approach:

```kotlin
@OptIn(ExperimentalCoroutinesApi::class)
class ProductListViewModelTest {

    // Set TestDispatcher for Coroutines
    private val testDispatcher = StandardTestDispatcher()
    
    // Mock Domain Use Cases
    private val getProductsUseCase: GetProductsUseCase = mockk()
    private val deleteProductUseCase: DeleteProductUseCase = mockk()

    private lateinit var viewModel: ProductListViewModel

    @BeforeEach
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
        
        // Setup initial default success responses
        coEvery { getProductsUseCase(any()) } returns flowOf(listOf(mockProduct(id = "p1")))
        
        viewModel = ProductListViewModel(getProductsUseCase, deleteProductUseCase)
    }

    @AfterEach
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `delete product intent updates state to loading then updates list successfully`() = runTest {
        // Arrange
        val productIdToDelete = "p1"
        coEvery { deleteProductUseCase(productIdToDelete) } returns Result.success(Unit)
        
        // Act & Assert using Turbine to observe UI State changes
        viewModel.uiState.test {
            // Initial State validation
            val initialState = awaitItem()
            assertFalse(initialState.isLoading)
            
            // Fire Delete Intent
            viewModel.handleIntent(ProductListIntent.DeleteProduct(productIdToDelete))
            
            // State should transition to loading
            val loadingState = awaitItem()
            assertTrue(loadingState.isLoading)
            
            // Finally, state transition on delete success (loading stops, product removed)
            val successState = awaitItem()
            assertFalse(successState.isLoading)
            assertTrue(successState.products.none { it.id == productIdToDelete })
            assertNull(successState.errorMessage)
            
            cancelAndConsumeRemainingEvents()
        }
        
        // Verify delete use case was executed exactly once
        coVerify(exactly = 1) { deleteProductUseCase(productIdToDelete) }
    }
}
```

---

## 7. Implementation Roadmap & Phases

Development is split into structured, testable iterations:

### Phase 1: Foundation & Core Infrastructure (Week 1)
*   Initialize project structure, Gradle build catalogs, and dependency configurations.
*   Setup Hilt DI modules, configure Apollo GraphQL client engine, and declare the local Room Database schema.
*   Write unit tests mapping remote GraphQL structures to clean local models.

### Phase 2: Auth, Onboarding & Store Creation (Week 2)
*   **TDD Focus**: Write use cases for `LoginUseCase` and `RegisterUseCase` with full error scenarios.
*   Implement authentication MVI flows (login form validation, token exchange, storing tokens in keystore).
*   Create adaptive Store Onboarding wizard for mobile and tablets.

### Phase 3: Product Catalog & Multi-Variant Editing (Week 3-4)
*   **TDD Focus**: Write tests verifying product validation logic (e.g., pricing constraints, empty field rejections).
*   Develop responsive product grids (2 to 5 columns depending on width classes).
*   Implement full create/edit panels using a double-column layout for tablets.
*   Integrate presigned media uploading flow.

### Phase 4: Orders & Order Fulfillment (Week 4-5)
*   **TDD Focus**: Test Order fulfillment state logic (verifying that items marked fulfilled update downstream states).
*   Implement **List-Detail Pane Scaffolding** for real-time order dashboard.
*   Hook up NATS/WebSocket subscriptions for incoming order notifications.

### Phase 5: Communications, Reviews & Polishing (Week 5-6)
*   Implement customer service chat dashboard (split-pane layout for tablets).
*   Add seller reviews & feedback metrics widget (DSR visualization cards).
*   Conduct edge-to-edge UI testing and memory leak audits using LeakCanary.
