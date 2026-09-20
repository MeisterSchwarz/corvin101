package discord

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hugolgst/rich-go/client"

	"corvin101/internal/data/zones"
	"corvin101/internal/game/state"
)

type Presence struct {
	mu sync.Mutex

	clientID string
	zones    *zones.Repository

	connected bool

	lastView viewState
	hasLast  bool
}

// viewState enthält ausschließlich die Teile des Game-States,
// die für die Discord Rich Presence relevant sind.
//
// Dadurch führt beispielsweise ein neuer Buff, Pip oder Spell
// nicht unnötig zu einem Discord-Update.
type viewState struct {
	Mode    state.Mode
	ZoneKey string
}

// New erstellt eine neue Discord Presence.
//
// Die Sprache wird nicht hier konfiguriert.
// Sie gehört bereits zum zones.Repository.
func New(
	clientID string,
	zoneRepository *zones.Repository,
) *Presence {
	return &Presence{
		clientID: clientID,
		zones:    zoneRepository,
	}
}

// Connect stellt die Verbindung zu Discord her.
//
// Mehrfache Aufrufe sind erlaubt.
func (p *Presence) Connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connected {
		return nil
	}

	if err := client.Login(p.clientID); err != nil {
		return fmt.Errorf(
			"discord login: %w",
			err,
		)
	}

	p.connected = true

	return nil
}

// Close trennt die Discord-Verbindung.
//
// Nach einem erneuten Connect wird die Presence beim nächsten
// Update garantiert neu gesetzt.
func (p *Presence) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.connected {
		return
	}

	client.Logout()

	p.connected = false
	p.hasLast = false
}

// Update synchronisiert die Discord Rich Presence mit einem
// Snapshot des zentralen Game-States.
//
// Discord kennt weder Parser noch Events. Der Snapshot ist die
// einzige Quelle für den aktuellen Spielzustand.
func (p *Presence) Update(
	ctx context.Context,
	snapshot state.Snapshot,
) {
	next := makeViewState(snapshot)

	p.mu.Lock()

	if !p.connected {
		p.mu.Unlock()
		return
	}

	if p.hasLast && p.lastView == next {
		p.mu.Unlock()
		return
	}

	p.mu.Unlock()

	// buildActivity kann über das ZoneRepository Daten laden.
	// Deshalb halten wir währenddessen bewusst keinen Mutex.
	activity, err := p.buildActivity(
		ctx,
		snapshot,
	)
	if err != nil {
		log.Println(
			"[discord] build activity:",
			err,
		)

		return
	}

	if err := client.SetActivity(activity); err != nil {
		log.Println(
			"[discord] set activity:",
			err,
		)

		return
	}

	// Erst nach erfolgreichem SetActivity merken wir uns den
	// Zustand als synchronisiert.
	p.mu.Lock()

	p.lastView = next
	p.hasLast = true

	p.mu.Unlock()
}

func makeViewState(
	snapshot state.Snapshot,
) viewState {
	return viewState{
		Mode:    snapshot.Game.Mode,
		ZoneKey: snapshot.Game.ZoneKey,
	}
}
