package datalayer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/dollarshaveclub/furan/pkg/models"
)

type FakeDataLayer struct {
	mtx           sync.RWMutex
	d             map[uuid.UUID]*models.Build
	apikeys       map[uuid.UUID]*models.APIKey
	listeners     map[uuid.UUID][]chan string
	cxllisteners  map[uuid.UUID][]chan struct{}
	runlisteners  map[uuid.UUID][]chan struct{}
	donelisteners map[uuid.UUID][]chan models.BuildStatus
}

var _ DataLayer = &FakeDataLayer{}

func (fdl *FakeDataLayer) init() {
	if fdl.d == nil {
		fdl.mtx.Lock()
		fdl.d = make(map[uuid.UUID]*models.Build)
		fdl.mtx.Unlock()
	}
	if fdl.apikeys == nil {
		fdl.mtx.Lock()
		fdl.apikeys = make(map[uuid.UUID]*models.APIKey)
		fdl.mtx.Unlock()
	}
	if fdl.listeners == nil {
		fdl.mtx.Lock()
		fdl.listeners = make(map[uuid.UUID][]chan string)
		fdl.mtx.Unlock()
	}
	if fdl.cxllisteners == nil {
		fdl.mtx.Lock()
		fdl.cxllisteners = make(map[uuid.UUID][]chan struct{})
		fdl.mtx.Unlock()
	}
	if fdl.runlisteners == nil {
		fdl.mtx.Lock()
		fdl.runlisteners = make(map[uuid.UUID][]chan struct{})
		fdl.mtx.Unlock()
	}
	if fdl.donelisteners == nil {
		fdl.mtx.Lock()
		fdl.donelisteners = make(map[uuid.UUID][]chan models.BuildStatus)
		fdl.mtx.Unlock()
	}
}

func (fdl *FakeDataLayer) CreateBuild(ctx context.Context, b models.Build) (uuid.UUID, error) {
	fdl.init()
	fdl.mtx.Lock()
	defer fdl.mtx.Unlock()
	b.ID = uuid.Must(uuid.NewV4())
	b.Status = models.BuildStatusNotStarted
	fdl.d[b.ID] = &b
	return b.ID, nil
}
func (fdl *FakeDataLayer) GetBuildByID(ctx context.Context, id uuid.UUID) (models.Build, error) {
	fdl.init()
	fdl.mtx.RLock()
	defer fdl.mtx.RUnlock()
	bsr, ok := fdl.d[id]
	if !ok {
		return models.Build{}, ErrNotFound
	}
	return *bsr, nil
}

func (fdl *FakeDataLayer) SetBuildCompletedTimestamp(ctx context.Context, id uuid.UUID, ts time.Time) error {
	fdl.init()
	fdl.mtx.Lock()
	defer fdl.mtx.Unlock()
	bsr, ok := fdl.d[id]
	if ok {
		bsr.Completed = ts
	}
	return nil
}

func (fdl *FakeDataLayer) SetBuildStatus(ctx context.Context, id uuid.UUID, s models.BuildStatus) error {
	fdl.init()
	fdl.mtx.Lock()
	defer fdl.mtx.Unlock()
	bsr, ok := fdl.d[id]
	if ok {
		bsr.Status = s
	}
	return nil
}

func (fdl *FakeDataLayer) DeleteBuild(ctx context.Context, id uuid.UUID) error {
	fdl.init()
	fdl.mtx.Lock()
	defer fdl.mtx.Unlock()
	delete(fdl.d, id)
	return nil
}

func (fdl *FakeDataLayer) ListenForBuildEvents(ctx context.Context, id uuid.UUID, c chan<- string) error {
	fdl.init()

	fdl.mtx.RLock()
	b, ok := fdl.d[id]
	if !ok {
		fdl.mtx.RUnlock()
		return fmt.Errorf("build not found")
	}
	fdl.mtx.RUnlock()

	if !b.CanAddEvent() {
		return fmt.Errorf("cannot add event to build with status %v", b.Status)
	}

	lc := make(chan string)

	fdl.mtx.Lock()
	fdl.listeners[id] = append(fdl.listeners[id], lc)
	i := len(fdl.listeners[id]) - 1 // index of listener
	fdl.mtx.Unlock()
	defer func() {
		// remove listener chan from listeners
		fdl.mtx.Lock()
		i = len(fdl.listeners[id]) - 1
		if i < 0 {
			fdl.mtx.Unlock()
			return
		}
		if i == 0 {
			delete(fdl.listeners, id)
			fdl.mtx.Unlock()
			return
		}
		fdl.listeners[id] = append(fdl.listeners[id][:i], fdl.listeners[id][i+1:]...)
		fdl.mtx.Unlock()
		close(lc)
	}()

	for {
		select {
		case e := <-lc:
			c <- e
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		}
	}
}

func (fdl *FakeDataLayer) AddEvent(ctx context.Context, id uuid.UUID, event string) error {
	fdl.init()

	fdl.mtx.Lock()
	b, ok := fdl.d[id]
	if !ok {
		fdl.mtx.Unlock()
		return nil
	}
	b.Events = append(fdl.d[id].Events, event)
	fdl.mtx.Unlock()

	fdl.mtx.RLock()
	defer fdl.mtx.RUnlock()

	for _, c := range fdl.listeners[id] {
		c <- event
	}

	return nil
}

func (fdl *FakeDataLayer) CancelBuild(ctx context.Context, id uuid.UUID) error {
	fdl.init()

	fdl.mtx.Lock()
	b, ok := fdl.d[id]
	if ok && b != nil {
		b.Status = models.BuildStatusCancelRequested
	}
	fdl.mtx.Unlock()

	fdl.mtx.RLock()
	defer fdl.mtx.RUnlock()

	for _, c := range fdl.cxllisteners[id] {
		c <- struct{}{}
	}

	return nil
}

func (fdl *FakeDataLayer) CancellationListeners() uint {
	fdl.init()
	fdl.mtx.RLock()
	defer fdl.mtx.RUnlock()
	return uint(len(fdl.cxllisteners))
}

func (fdl *FakeDataLayer) ListenForCancellation(ctx context.Context, id uuid.UUID) error {
	fdl.init()

	fdl.mtx.RLock()
	b, ok := fdl.d[id]
	if !ok {
		fdl.mtx.RUnlock()
		return fmt.Errorf("build not found")
	}
	fdl.mtx.RUnlock()

	if !b.Running() {
		return fmt.Errorf("cannot cxl build with status %v", b.Status)
	}

	lc := make(chan struct{})

	fdl.mtx.Lock()
	fdl.cxllisteners[id] = append(fdl.cxllisteners[id], lc)
	i := len(fdl.cxllisteners[id]) - 1 // index of listener
	fdl.mtx.Unlock()
	defer func() {
		// remove listener chan from cxllisteners
		fdl.mtx.Lock()
		i = len(fdl.cxllisteners[id]) - 1
		if i < 0 {
			fdl.mtx.Unlock()
			return
		}
		if i == 0 {
			delete(fdl.cxllisteners, id)
			fdl.mtx.Unlock()
			return
		}
		fdl.cxllisteners[id] = append(fdl.cxllisteners[id][:i], fdl.cxllisteners[id][i+1:]...)
		fdl.mtx.Unlock()
		close(lc)
	}()

	select {
	case <-lc:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

func (fdl *FakeDataLayer) SetBuildAsRunning(ctx context.Context, id uuid.UUID) error {
	fdl.init()

	fdl.mtx.Lock()
	b, ok := fdl.d[id]
	if ok && b != nil {
		b.Status = models.BuildStatusRunning
	}
	fdl.mtx.Unlock()

	fdl.mtx.RLock()
	defer fdl.mtx.RUnlock()

	for _, c := range fdl.runlisteners[id] {
		c <- struct{}{}
	}

	return nil
}

func (fdl *FakeDataLayer) ListenForBuildRunning(ctx context.Context, id uuid.UUID) error {
	fdl.init()

	fdl.mtx.RLock()
	b, ok := fdl.d[id]
	if !ok {
		fdl.mtx.RUnlock()
		return fmt.Errorf("build not found")
	}
	fdl.mtx.RUnlock()

	if b.Status != models.BuildStatusNotStarted {
		return fmt.Errorf("bad build status: %v", b.Status)
	}

	lc := make(chan struct{})

	fdl.mtx.Lock()
	fdl.runlisteners[id] = append(fdl.runlisteners[id], lc)
	i := len(fdl.runlisteners[id]) - 1 // index of listener
	fdl.mtx.Unlock()
	defer func() {
		// remove listener chan from runlisteners
		fdl.mtx.Lock()
		i = len(fdl.runlisteners[id]) - 1
		if i < 0 {
			fdl.mtx.Unlock()
			return
		}
		if i == 0 {
			delete(fdl.runlisteners, id)
			fdl.mtx.Unlock()
			return
		}
		fdl.runlisteners[id] = append(fdl.runlisteners[id][:i], fdl.runlisteners[id][i+1:]...)
		fdl.mtx.Unlock()
		close(lc)
	}()

	select {
	case <-lc:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

func (fdl *FakeDataLayer) SetBuildAsCompleted(ctx context.Context, id uuid.UUID, status models.BuildStatus) error {
	fdl.init()

	fdl.mtx.Lock()
	b, ok := fdl.d[id]
	if ok && b != nil {
		b.Status = status
	}
	fdl.mtx.Unlock()

	fdl.mtx.RLock()
	defer fdl.mtx.RUnlock()

	for _, c := range fdl.donelisteners[id] {
		c <- status
	}

	return nil
}

func (fdl *FakeDataLayer) ListenForBuildCompleted(ctx context.Context, id uuid.UUID) (models.BuildStatus, error) {
	fdl.init()

	fdl.mtx.RLock()
	b, ok := fdl.d[id]
	if !ok {
		fdl.mtx.RUnlock()
		return 0, fmt.Errorf("build not found")
	}
	// if build is already finished, return status
	if b.Status.TerminalState() {
		status := b.Status
		fdl.mtx.RUnlock()
		return status, nil
	}
	fdl.mtx.RUnlock()

	lc := make(chan models.BuildStatus)

	fdl.mtx.Lock()
	fdl.donelisteners[id] = append(fdl.donelisteners[id], lc)
	i := len(fdl.donelisteners[id]) - 1 // index of listener
	fdl.mtx.Unlock()
	defer func() {
		// remove listener chan from donelisteners
		fdl.mtx.Lock()
		i = len(fdl.donelisteners[id]) - 1
		if i < 0 {
			fdl.mtx.Unlock()
			return
		}
		if i == 0 {
			delete(fdl.donelisteners, id)
			fdl.mtx.Unlock()
			return
		}
		fdl.donelisteners[id] = append(fdl.donelisteners[id][:i], fdl.donelisteners[id][i+1:]...)
		fdl.mtx.Unlock()
		close(lc)
	}()

	select {
	case s := <-lc:
		return s, nil
	case <-ctx.Done():
		return 0, fmt.Errorf("context cancelled")
	}
}

func (fdl *FakeDataLayer) CreateAPIKey(ctx context.Context, ak models.APIKey) (uuid.UUID, error) {
	fdl.init()
	fdl.mtx.Lock()
	defer fdl.mtx.Unlock()
	ak.ID = uuid.Must(uuid.NewV4())
	fdl.apikeys[ak.ID] = &ak
	return ak.ID, nil
}

func (fdl *FakeDataLayer) GetAPIKey(ctx context.Context, id uuid.UUID) (models.APIKey, error) {
	fdl.init()
	fdl.mtx.RLock()
	defer fdl.mtx.RUnlock()
	apk, ok := fdl.apikeys[id]
	if !ok {
		return models.APIKey{}, ErrNotFound
	}
	return *apk, nil
}

func (fdl *FakeDataLayer) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	fdl.init()
	fdl.mtx.Lock()
	defer fdl.mtx.Unlock()
	delete(fdl.apikeys, id)
	return nil
}
