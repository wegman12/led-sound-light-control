# Implementation Plan: Audio Configuration Management System

## Development Guidelines

**IMPORTANT**: This is a multi-phase implementation. To enable easy rollback if issues arise:
- Commit after completing each file or logical unit of work
- Use descriptive commit messages referencing the phase (e.g., "Phase 2: Add config repository API endpoints")
- Test each phase before moving to the next
- If you need to stop and resume in a new context, refer back to this plan

---

## Executive Summary

This plan implements a disk-based audio configuration management system that allows users to:
1. Store multiple audio tuning configurations on disk
2. Set an "active" configuration that persists across restarts
3. Manage configurations through a new unified "Audio Configuration" page
4. Have the Audio Lights page auto-populate with the active configuration's values

The configuration contains **tuning parameters only** (frequency cutoffs, scaling factors, noise thresholds, smoothing per band). Color-to-band mappings remain in the Audio Lights page.

---

## Architecture Overview

### Current State
```
Audio Tuning Storage:
/etc/led-sound-light-control/audio_tuning_config.json  (single file)
                    ↓
              AudioTuningConfigManager (load/save single config)
                    ↓
              HTTP API (GET/PUT/POST save)
                    ↓
              Audio Tuning Page (edit single config)
```

### Proposed State
```
Audio Configuration Repository:
/etc/led-sound-light-control/audio-configs/
├── active.json           (pointer to current config name)
├── default.json          (built-in default config)
├── party-mode.json       (user-created)
├── quiet-listening.json  (user-created)
└── ...

              ↓
      AudioConfigRepository (manage multiple configs)
              ↓
      HTTP API (list/get/create/update/delete/set-active)
              ↓
      Audio Configuration Page (unified management)
              ↓
      Audio Lights Page (auto-populates from active config)
```

---

## Core Design Decisions

### 1. Configuration Scope (User Decision: Tuning Only)
Each configuration file contains **AudioTuningConfig** only:
- Frequency band cutoffs (bass_cutoff, mid_high_cutoff, treble_cutoff)
- Per-band settings: scaling_factor, noise_threshold, min_power_value, max_power_value, smoothing

Color-to-band mappings (which LED color responds to which frequency band) remain in the Audio Lights page UI.

### 2. Active Configuration (User Decision: Auto-apply Defaults)
- The Audio Lights page pre-populates with values from the active configuration
- Users can override values temporarily in the Audio Lights page
- Overrides are NOT automatically saved back to the config
- Users must explicitly save changes via the Audio Configuration page

### 3. Visualization (User Decision: Real-time Only)
- The Audio Configuration page shows real-time streaming visualization
- No CSV-based simulation (Audio Visualizer page will be fully removed)

### 4. Storage Location
```
/etc/led-sound-light-control/audio-configs/
```
This directory contains:
- Individual config JSON files named by user (e.g., `party-mode.json`)
- A special `_active.json` file that stores the name of the currently active config

### 5. Page Consolidation
- **Delete completely**: Audio Visualizer page (remove from codebase)
- **Remove from navigation**: Audio Tuning page (redirect to new page, keep code temporarily)
- **Add**: Audio Configuration page (via Settings dropdown in header)
- **Keep**: Audio Lights page (uses active config as defaults)

### 6. Default Configuration
- Ship with a single "Default" configuration only
- Users create environment-specific configs as needed (Party Mode, etc.)

---

## Data Models

### AudioTuningConfig (Existing - No Changes)
```go
// File: api/light/audio_tuning_config.go
type AudioTuningConfig struct {
    BassCutoff    int              `json:"bass_cutoff"`
    MidHighCutoff int              `json:"mid_high_cutoff"`
    TrebleCutoff  int              `json:"treble_cutoff"`
    Bass          BandTuningConfig `json:"bass"`
    MidLow        BandTuningConfig `json:"mid_low"`
    MidHigh       BandTuningConfig `json:"mid_high"`
    Treble        BandTuningConfig `json:"treble"`
}

type BandTuningConfig struct {
    ScalingFactor  float64 `json:"scaling_factor"`
    NoiseThreshold float64 `json:"noise_threshold"`
    MinPowerValue  float64 `json:"min_power_value"`
    MaxPowerValue  float64 `json:"max_power_value"`
    Smoothing      float64 `json:"smoothing"`
}
```

### SavedAudioConfig (New)
```go
// File: api/light/audio_config_repository.go
type SavedAudioConfig struct {
    Name        string            `json:"name"`        // Unique identifier (filename without .json)
    DisplayName string            `json:"display_name"` // Human-readable name
    Description string            `json:"description"` // Optional description
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
    Config      AudioTuningConfig `json:"config"`
}
```

### ActiveConfigPointer (New)
```go
// File: api/light/audio_config_repository.go
// Stored in _active.json
type ActiveConfigPointer struct {
    ConfigName string    `json:"config_name"` // Name of the active config
    SetAt      time.Time `json:"set_at"`
}
```

---

## API Endpoints

### New Endpoints for Configuration Repository

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/audio/configs` | List all saved configurations |
| GET | `/api/audio/configs/:name` | Get a specific configuration by name |
| POST | `/api/audio/configs` | Create a new configuration |
| PUT | `/api/audio/configs/:name` | Update an existing configuration |
| DELETE | `/api/audio/configs/:name` | Delete a configuration |
| GET | `/api/audio/configs/active` | Get the currently active configuration |
| PUT | `/api/audio/configs/active` | Set a configuration as active |

### Existing Endpoints (Behavior Changes)

| Endpoint | Change |
|----------|--------|
| `GET /api/audio/tuning/config` | Returns the **active** configuration's tuning values |
| `PUT /api/audio/tuning/config` | Updates the **in-memory** tuning (temporary override) |
| `POST /api/audio/tuning/config/save` | **Deprecated** - Use PUT `/api/audio/configs/:name` instead |
| `WS /api/audio/tuning/stream` | No change - streams real-time audio data |
| `GET /api/lights/audio/config` | Returns active config's band settings (for Audio Lights page defaults) |

### API Response Examples

**GET /api/audio/configs**
```json
{
  "configs": [
    {
      "name": "default",
      "display_name": "Default",
      "description": "Built-in default configuration",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    },
    {
      "name": "party-mode",
      "display_name": "Party Mode",
      "description": "High sensitivity for loud music",
      "created_at": "2025-12-20T10:30:00Z",
      "updated_at": "2025-12-20T14:15:00Z"
    }
  ],
  "active_config": "party-mode"
}
```

**GET /api/audio/configs/active**
```json
{
  "name": "party-mode",
  "display_name": "Party Mode",
  "description": "High sensitivity for loud music",
  "created_at": "2025-12-20T10:30:00Z",
  "updated_at": "2025-12-20T14:15:00Z",
  "config": {
    "bass_cutoff": 100,
    "mid_high_cutoff": 500,
    "treble_cutoff": 2000,
    "bass": {
      "scaling_factor": 0.000001,
      "noise_threshold": 92565436,
      "min_power_value": 0.0,
      "max_power_value": 1.0,
      "smoothing": 0.3
    },
    ...
  }
}
```

**PUT /api/audio/configs/active**
Request body:
```json
{
  "config_name": "default"
}
```

**POST /api/audio/configs**
Request body:
```json
{
  "name": "quiet-listening",
  "display_name": "Quiet Listening",
  "description": "Lower sensitivity for quiet ambient music",
  "config": { ... }
}
```

---

## Website Changes

### Header Component Changes

**Current Header:**
```
[LED Manager]                    [On] [Off]
```

**Proposed Header:**
```
[LED Manager]                    [On] [Off] [Settings ▼]
                                              └─ Audio Configuration
```

Implementation:
- Add MUI Menu component triggered by "Settings" button
- Single menu item initially: "Audio Configuration"
- Navigates to `/audio-configuration`

### Route Changes

**Remove from navigation** (but keep routes for backward compatibility):
- `/audio-visualizer` → Redirect to `/audio-configuration`
- `/audio-tuning` → Redirect to `/audio-configuration`

**Add new route:**
- `/audio-configuration` → AudioConfigurationPage component

**Keep unchanged:**
- `/audio-lights` → AudioLightsPage (with modifications to load active config)

### Dashboard Page Changes

Remove cards for:
- Audio Visualizer
- Audio Tuning

The Audio Configuration is now accessed via Settings dropdown, not dashboard.

### New Audio Configuration Page

**Layout Structure:**
```
┌─────────────────────────────────────────────────────────────────┐
│ Audio Configuration                                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Saved Configurations                                      │   │
│  │ ┌────────────────┬────────────────┬────────────────────┐ │   │
│  │ │ ● Default      │ ○ Party Mode   │ ○ Quiet Listening  │ │   │
│  │ │   (active)     │   [Edit][Delete] │  [Edit][Delete]   │ │   │
│  │ └────────────────┴────────────────┴────────────────────┘ │   │
│  │                                    [+ New Configuration] │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Configuration Editor: Party Mode                          │   │
│  │ ┌───────────────────────────────────────────────────────┐ │   │
│  │ │ Display Name: [Party Mode          ]                   │ │   │
│  │ │ Description:  [High sensitivity for loud music    ]    │ │   │
│  │ └───────────────────────────────────────────────────────┘ │   │
│  │                                                           │   │
│  │ Frequency Band Cutoffs                                    │   │
│  │ Bass Cutoff: [100] Hz  Mid-High: [500] Hz  Treble: [2000] │   │
│  │                                                           │   │
│  │ Band Settings                                             │   │
│  │ ┌─────────┬─────────┬─────────┬─────────┐                 │   │
│  │ │  Bass   │ Mid-Low │Mid-High │ Treble  │                 │   │
│  │ ├─────────┼─────────┼─────────┼─────────┤                 │   │
│  │ │Scale: _ │Scale: _ │Scale: _ │Scale: _ │                 │   │
│  │ │Noise: _ │Noise: _ │Noise: _ │Noise: _ │                 │   │
│  │ │Min: _   │Min: _   │Min: _   │Min: _   │                 │   │
│  │ │Max: _   │Max: _   │Max: _   │Max: _   │                 │   │
│  │ │Smooth: _│Smooth: _│Smooth: _│Smooth: _│                 │   │
│  │ └─────────┴─────────┴─────────┴─────────┘                 │   │
│  │                                                           │   │
│  │ [Save Changes] [Set as Active] [Discard Changes]          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Real-Time Audio Preview              [● Streaming] [Stop] │   │
│  │ ┌───────────────────────────────────────────────────────┐ │   │
│  │ │ [Raw Audio Chart - 30 second rolling window]          │ │   │
│  │ └───────────────────────────────────────────────────────┘ │   │
│  │ ┌───────────────────────────────────────────────────────┐ │   │
│  │ │ [Processed LED Power Chart - shows effect of config]  │ │   │
│  │ └───────────────────────────────────────────────────────┘ │   │
│  │ Toggle: [Raw] [Processed] [Both]                          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key Features:**
1. **Configuration List**: Radio buttons showing all saved configs, active one marked
2. **Configuration Editor**: Edit selected config's parameters
3. **Real-Time Preview**: WebSocket streaming to visualize how current settings affect audio
4. **Actions**: Save, Set as Active, Delete, Create New

### Audio Lights Page Changes

**Current Behavior:**
- User manually configures all parameters for each color
- No persistence between visits

**New Behavior:**
1. On page load, fetch active configuration via `GET /api/audio/configs/active`
2. Pre-populate all band parameters (scaling, noise threshold, min/max power, smoothing) from active config
3. User can still override any value (changes are temporary)
4. Show indicator: "Using: [Active Config Name] (with overrides)" when values differ from active config
5. Add link: "Manage configurations →" to navigate to Audio Configuration page

---

## Implementation Phases

### Phase 1: API - Configuration Repository Foundation

**Goal**: Create the backend infrastructure for managing multiple configs.

**Files to create:**
1. `api/light/audio_config_repository.go`
   - `AudioConfigRepository` struct
   - `NewAudioConfigRepository(basePath string)` constructor
   - `List() ([]SavedAudioConfig, error)` - list all configs
   - `Get(name string) (*SavedAudioConfig, error)` - get single config
   - `Create(config SavedAudioConfig) error` - create new config
   - `Update(name string, config SavedAudioConfig) error` - update existing
   - `Delete(name string) error` - delete config
   - `GetActive() (*SavedAudioConfig, error)` - get active config
   - `SetActive(name string) error` - set active config
   - `EnsureDefaultExists()` - create default config if none exist

2. `api/light/audio_config_repository_test.go`
   - Unit tests for all repository methods
   - Test file system operations
   - Test validation (prevent delete of active config, etc.)

**Files to modify:**
1. `api/light/audio_tuning_config.go`
   - Update `LoadConfig()` to load from active config in repository
   - Keep `GetConfig()` and `UpdateConfig()` for in-memory operations

**Commit after**: Repository implementation complete with tests

---

### Phase 2: API - HTTP Endpoints for Configuration Management

**Goal**: Expose configuration repository via HTTP API.

**Files to create:**
1. `api/light/audio_config_handler.go`
   - `HandleListConfigs` - GET /api/audio/configs
   - `HandleGetConfig` - GET /api/audio/configs/:name
   - `HandleCreateConfig` - POST /api/audio/configs
   - `HandleUpdateConfig` - PUT /api/audio/configs/:name
   - `HandleDeleteConfig` - DELETE /api/audio/configs/:name
   - `HandleGetActiveConfig` - GET /api/audio/configs/active
   - `HandleSetActiveConfig` - PUT /api/audio/configs/active

**Files to modify:**
1. `api/light/routes.go`
   - Register new configuration endpoints
   - Inject repository dependency

2. `api/infrastructure/server/routes.go` (if needed)
   - Wire up repository to server initialization

**Commit after**: All endpoints implemented and manually tested

---

### Phase 3: API - Integrate Active Config with Existing Endpoints

**Goal**: Make existing audio tuning endpoints use the active configuration.

**Files to modify:**
1. `api/light/audio_tuning_config.go`
   - Inject repository reference
   - `LoadConfig()` loads from active config in repository
   - `SaveConfig()` updates the active config in repository

2. `api/audio/tuning_handler.go` (if exists)
   - Ensure GET /api/audio/tuning/config returns active config's values
   - PUT /api/audio/tuning/config updates in-memory only (temporary)

3. `api/light/audio_handler.go`
   - GET /api/lights/audio/config returns active config's band settings

**Commit after**: Existing endpoints work with new repository

---

### Phase 4: Website - Header Settings Dropdown

**Goal**: Add Settings dropdown to header with Audio Configuration link.

**Files to modify:**
1. `website/src/components/Header.tsx`
   - Add Settings button with MUI Menu
   - Add "Audio Configuration" menu item
   - Navigate to `/audio-configuration` on click

**Files to create:**
1. `website/src/components/SettingsMenu.tsx` (optional - could be inline)
   - Reusable settings menu component

**Commit after**: Header has working Settings dropdown

---

### Phase 5: Website - API Services for Configuration Management

**Goal**: Add TypeScript services and types for new API endpoints.

**Files to modify:**
1. `website/src/types/api.ts`
   - Add `SavedAudioConfig` interface
   - Add `ActiveConfigPointer` interface
   - Add `ConfigListResponse` interface

2. `website/src/services/audioConfig.ts` (new file)
   - `listConfigs()` - GET /api/audio/configs
   - `getConfig(name)` - GET /api/audio/configs/:name
   - `createConfig(config)` - POST /api/audio/configs
   - `updateConfig(name, config)` - PUT /api/audio/configs/:name
   - `deleteConfig(name)` - DELETE /api/audio/configs/:name
   - `getActiveConfig()` - GET /api/audio/configs/active
   - `setActiveConfig(name)` - PUT /api/audio/configs/active

3. `website/src/services/index.ts`
   - Export new audioConfig service

**Commit after**: Services complete with proper typing

---

### Phase 6: Website - Audio Configuration Page (Core)

**Goal**: Create the new Audio Configuration page with config management.

**Files to create:**
1. `website/src/pages/AudioConfigurationPage.tsx`
   - Configuration list with radio buttons
   - Configuration editor form
   - Create/Update/Delete functionality
   - Set as Active button
   - Form validation

2. `website/src/components/ConfigurationCard.tsx` (optional)
   - Reusable card for displaying a single configuration

3. `website/src/components/BandSettingsEditor.tsx` (optional)
   - Reusable editor for band-specific settings (scaling, noise, etc.)

**Files to modify:**
1. `website/src/routes/AppRoutes.tsx`
   - Add route for `/audio-configuration`

**Commit after**: Basic page working with CRUD operations

---

### Phase 7: Website - Audio Configuration Page (Real-Time Preview)

**Goal**: Add real-time audio streaming visualization to the configuration page.

**Files to modify:**
1. `website/src/pages/AudioConfigurationPage.tsx`
   - Add WebSocket connection for streaming
   - Add raw/processed audio charts (reuse logic from AudioTuningPage)
   - Add start/stop streaming controls
   - Show how current editor values affect processed output

**Implementation notes:**
- Extract chart logic from AudioTuningPage into reusable hooks/components
- `website/src/hooks/useAudioStream.ts` - WebSocket streaming hook
- `website/src/components/AudioStreamChart.tsx` - Reusable chart component

**Commit after**: Real-time preview working

---

### Phase 8: Website - Audio Lights Page Integration

**Goal**: Make Audio Lights page load defaults from active configuration.

**Files to modify:**
1. `website/src/pages/AudioLightsPage.tsx`
   - On mount, fetch active config via `getActiveConfig()`
   - Pre-populate band settings for each color from active config
   - Add indicator showing which config is loaded
   - Add "Manage configurations" link to Audio Configuration page
   - Track if user has overridden any values (show "modified" indicator)

**Commit after**: Audio Lights loads active config defaults

---

### Phase 9: Website - Navigation Cleanup & Page Removal

**Goal**: Remove old pages, clean up navigation.

**Files to delete:**
1. `website/src/pages/AudioVisualizerPage.tsx` - Remove completely

**Files to modify:**
1. `website/src/pages/DashboardPage.tsx`
   - Remove Audio Visualizer card
   - Remove Audio Tuning card
   - (Keep Audio Lights card)

2. `website/src/routes/AppRoutes.tsx`
   - Remove `/audio-visualizer` route entirely
   - Add redirect from `/audio-tuning` to `/audio-configuration`

**Commit after**: Navigation cleaned up, Audio Visualizer removed

---

### Phase 10: Testing and Polish

**Goal**: Ensure everything works together, add error handling.

**Tasks:**
1. End-to-end testing of all flows
2. Error handling for edge cases:
   - No configurations exist (create default)
   - Active config deleted (fall back to default)
   - Network errors during save
3. Loading states and user feedback
4. Mobile responsiveness of new page

**Commit after**: All testing complete

---

## File Summary

### New Files (API - 3 files)
1. `api/light/audio_config_repository.go` - Repository for multiple configs
2. `api/light/audio_config_repository_test.go` - Repository tests
3. `api/light/audio_config_handler.go` - HTTP handlers for config management

### Modified Files (API - 3-4 files)
1. `api/light/routes.go` - Register new endpoints
2. `api/light/audio_tuning_config.go` - Integrate with repository
3. `api/light/audio_handler.go` - Return active config defaults
4. `api/infrastructure/server/routes.go` - Wire dependencies (if needed)

### New Files (Website - 3-5 files)
1. `website/src/pages/AudioConfigurationPage.tsx` - Main new page
2. `website/src/services/audioConfig.ts` - API services
3. `website/src/hooks/useAudioStream.ts` - Reusable streaming hook
4. `website/src/components/AudioStreamChart.tsx` - Reusable chart (optional)
5. `website/src/components/BandSettingsEditor.tsx` - Reusable editor (optional)

### Deleted Files (Website - 1 file)
1. `website/src/pages/AudioVisualizerPage.tsx` - Remove completely

### Modified Files (Website - 6 files)
1. `website/src/components/Header.tsx` - Add Settings dropdown
2. `website/src/types/api.ts` - Add new types
3. `website/src/services/index.ts` - Export new service
4. `website/src/routes/AppRoutes.tsx` - Add new route, remove Audio Visualizer route
5. `website/src/pages/DashboardPage.tsx` - Remove old cards
6. `website/src/pages/AudioLightsPage.tsx` - Load active config defaults

---

## Validation Rules

### Configuration Name Validation
- Must be unique (case-insensitive)
- Must be valid filename (alphanumeric, hyphens, underscores)
- Cannot be `_active` (reserved)
- Length: 1-50 characters

### Cannot Delete Active Configuration
- API returns 400 error if attempting to delete the active config
- User must first set a different config as active

### Default Configuration
- A `default` configuration is created on first run
- Contains the current hardcoded default values
- Cannot be deleted (protected)

---

## Migration Strategy

### First Run After Update
1. Check if `/etc/led-sound-light-control/audio-configs/` exists
2. If not, create directory
3. Check if `_active.json` exists
4. If not:
   - If old `/etc/led-sound-light-control/audio_tuning_config.json` exists:
     - Migrate it to `audio-configs/migrated.json`
     - Set as active
   - If no old config:
     - Create `default.json` with hardcoded defaults
     - Set as active

### Backward Compatibility
- Keep old `audio_tuning_config.json` location working for read (fallback)
- New writes always go to repository

---

## Current Status

**Last Updated**: 2025-12-20

**Phase Status**:
- [x] Phase 1: API - Configuration Repository Foundation
- [x] Phase 2: API - HTTP Endpoints for Configuration Management
- [x] Phase 3: API - Integrate Active Config with Existing Endpoints
- [ ] Phase 4: Website - Header Settings Dropdown
- [ ] Phase 5: Website - API Services for Configuration Management
- [ ] Phase 6: Website - Audio Configuration Page (Core)
- [ ] Phase 7: Website - Audio Configuration Page (Real-Time Preview)
- [ ] Phase 8: Website - Audio Lights Page Integration
- [ ] Phase 9: Website - Navigation Cleanup & Page Removal
- [ ] Phase 10: Testing and Polish

**Notes**: API phases 1-3 complete. Starting website implementation.

---

## Open Questions / Future Considerations

1. **Import/Export**: Should users be able to export configs as JSON and import them?
2. **Config Sharing**: Cloud sync or sharing between devices?
3. **Undo**: Should there be an undo/history for config changes?

## Resolved Decisions

- **Audio Visualizer Page**: Fully remove from codebase (not just hide)
- **Default Presets**: Single "Default" config only; users create environment-specific configs
- **Config Storage**: `/etc/led-sound-light-control/audio-configs/` confirmed
