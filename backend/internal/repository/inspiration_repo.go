package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type inspirationRepository struct {
	db *sql.DB
}

// NewInspirationRepository creates the 灵感库 store (raw SQL over lib_poems / lib_songs).
func NewInspirationRepository(db *sql.DB) service.InspirationRepository {
	return &inspirationRepository{db: db}
}

// libPoemText is the searched text; it matches the trigram index of migration 314.
const libPoemText = `(title || ' ' || author || ' ' || content)`

const libPoemColumns = `id, collection, title, author, dynasty, content, notes, tags, famous, popularity, starred`

func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

type sqlArgs struct{ list []any }

func (a *sqlArgs) add(v any) string {
	a.list = append(a.list, v)
	return "$" + strconv.Itoa(len(a.list))
}

func libPoemFilters(q service.LibPoemQuery, terms []string, a *sqlArgs) []string {
	var where []string
	for _, t := range terms {
		where = append(where, libPoemText+" ILIKE "+a.add("%"+likeEscape(t)+"%"))
	}
	if q.Dynasty != "" {
		where = append(where, "dynasty = "+a.add(q.Dynasty))
	}
	if q.Collection != "" {
		where = append(where, "collection = "+a.add(q.Collection))
	}
	if q.Author != "" {
		where = append(where, "author = "+a.add(q.Author))
	}
	if q.Famous {
		where = append(where, "famous")
	}
	if q.Starred {
		where = append(where, "starred")
	}
	return where
}

func scanLibPoem(row interface{ Scan(...any) error }, extra ...any) (*service.LibPoem, error) {
	var p service.LibPoem
	var tags pq.StringArray
	dest := append([]any{&p.ID, &p.Collection, &p.Title, &p.Author, &p.Dynasty, &p.Content, &p.Notes, &tags, &p.Famous, &p.Popularity, &p.Starred}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	p.Tags = []string(tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return &p, nil
}

func (r *inspirationRepository) queryPoems(ctx context.Context, query string, args []any, score bool) ([]service.LibPoem, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LibPoem{}
	for rows.Next() {
		var n int
		var extra []any
		if score {
			extra = append(extra, &n)
		}
		p, err := scanLibPoem(rows, extra...)
		if err != nil {
			return nil, err
		}
		p.Score = n
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *inspirationRepository) SearchPoems(ctx context.Context, q service.LibPoemQuery) ([]service.LibPoem, bool, error) {
	a := &sqlArgs{}
	where := libPoemFilters(q, libQueryTerms(q.Q), a)
	query := `SELECT ` + libPoemColumns + ` FROM lib_poems`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	if q.Random {
		query += " ORDER BY random()"
	} else {
		query += " ORDER BY famous DESC, popularity DESC, id"
	}
	query += " LIMIT " + a.add(q.Limit+1) + " OFFSET " + a.add(q.Offset)
	poems, err := r.queryPoems(ctx, query, a.list, false)
	if err != nil {
		return nil, false, err
	}
	more := len(poems) > q.Limit
	if more {
		poems = poems[:q.Limit]
	}
	return poems, more, nil
}

// libQueryTerms mirrors the service's word split (the query struct keeps its terms private).
func libQueryTerms(q string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(q, func(r rune) bool { return r == ' ' || r == '　' || r == ',' || r == '，' || r == '、' }) {
		if f = strings.TrimSpace(f); f != "" && len(out) < 8 {
			out = append(out, f)
		}
	}
	return out
}

func (r *inspirationRepository) GetPoem(ctx context.Context, id int64) (*service.LibPoem, error) {
	p, err := scanLibPoem(r.db.QueryRowContext(ctx, `SELECT `+libPoemColumns+` FROM lib_poems WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (r *inspirationRepository) StarPoem(ctx context.Context, id int64, starred bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE lib_poems SET starred = $2 WHERE id = $1`, id, starred)
	return err
}

func (r *inspirationRepository) PoemCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT collection, COUNT(*) FROM lib_poems GROUP BY collection`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var c string
		var n int
		if err := rows.Scan(&c, &n); err != nil {
			return nil, err
		}
		out[c] = n
	}
	return out, rows.Err()
}

func (r *inspirationRepository) FindPoems(ctx context.Context, words []string, q service.LibPoemQuery, limit int) ([]service.LibPoem, error) {
	if len(words) == 0 {
		return []service.LibPoem{}, nil
	}
	a := &sqlArgs{}
	patterns := make([]string, len(words))
	var score []string
	for i, w := range words {
		patterns[i] = "%" + likeEscape(w) + "%"
		score = append(score, "(CASE WHEN "+libPoemText+" LIKE "+a.add(patterns[i])+" THEN 1 ELSE 0 END)")
	}
	where := append(libPoemFilters(q, nil, a), libPoemText+" LIKE ANY("+a.add(pq.Array(patterns))+")")
	query := `SELECT ` + libPoemColumns + `, ` + strings.Join(score, " + ") + ` AS score FROM lib_poems WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY score DESC, famous DESC, popularity DESC, length(content), id LIMIT ` + a.add(limit)
	return r.queryPoems(ctx, query, a.list, true)
}

func (r *inspirationRepository) FindPoemByTitle(ctx context.Context, title, author string) (*service.LibPoem, error) {
	title, author = strings.TrimSpace(title), strings.TrimSpace(author)
	if title == "" {
		return nil, nil
	}
	// The exact title first, then titles that start with it (「饮酒」 → 「饮酒二十首其五」).
	for _, cond := range []struct{ sql, v string }{{`title = $1`, title}, {`title LIKE $1`, likeEscape(title) + "%"}} {
		p, err := scanLibPoem(r.db.QueryRowContext(ctx, `SELECT `+libPoemColumns+` FROM lib_poems WHERE `+cond.sql+
			` AND ($2 = '' OR author = $2) ORDER BY famous DESC, popularity DESC, id LIMIT 1`, cond.v, author))
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return nil, nil
}

var libNonHanGap = `[^㐀-鿿]{0,3}`

func (r *inspirationRepository) PoemsContain(ctx context.Context, han string) (bool, error) {
	if han == "" {
		return false, nil
	}
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM lib_poems WHERE `+libPoemText+` LIKE $1)`, "%"+likeEscape(han)+"%").Scan(&ok)
	if err != nil || ok {
		return ok, err
	}
	// The run may cross a comma between two half-lines.
	parts := make([]string, 0, len(han))
	for _, c := range han {
		parts = append(parts, regexp.QuoteMeta(string(c)))
	}
	err = r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM lib_poems WHERE content ~ $1)`, strings.Join(parts, libNonHanGap)).Scan(&ok)
	return ok, err
}

// ---- songs ----

const libSongColumns = `id, title, artist, lyricist, composer, year, album, lyrics, summary, imagery, moods, tags, allusions,
       notes, link_url, starred, created_by, created_at, updated_at`

func scanLibSong(row interface{ Scan(...any) error }) (*service.LibSong, error) {
	var s service.LibSong
	var year sql.NullInt64
	var createdBy sql.NullInt64
	var imagery, moods, tags pq.StringArray
	var allusions []byte
	if err := row.Scan(&s.ID, &s.Title, &s.Artist, &s.Lyricist, &s.Composer, &year, &s.Album, &s.Lyrics, &s.Summary,
		&imagery, &moods, &tags, &allusions, &s.Notes, &s.LinkURL, &s.Starred, &createdBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	if year.Valid {
		y := int(year.Int64)
		s.Year = &y
	}
	if createdBy.Valid {
		s.CreatedBy = &createdBy.Int64
	}
	s.Imagery, s.Moods, s.Tags = nonNilStrings([]string(imagery)), nonNilStrings([]string(moods)), nonNilStrings([]string(tags))
	if err := json.Unmarshal(allusions, &s.Allusions); err != nil || s.Allusions == nil {
		s.Allusions = []service.LibAllusion{}
	}
	return &s, nil
}

func (r *inspirationRepository) ListSongs(ctx context.Context, q service.LibSongQuery) ([]service.LibSong, int, error) {
	a := &sqlArgs{}
	var where []string
	for _, t := range libQueryTerms(q.Q) {
		p := a.add("%" + likeEscape(t) + "%")
		where = append(where, "(title ILIKE "+p+" OR artist ILIKE "+p+" OR lyricist ILIKE "+p+" OR lyrics ILIKE "+p+
			" OR summary ILIKE "+p+" OR notes ILIKE "+p+" OR array_to_string(imagery || moods || tags, ' ') ILIKE "+p+")")
	}
	if q.Artist != "" {
		where = append(where, "artist = "+a.add(q.Artist))
	}
	if q.Tag != "" {
		p := a.add(q.Tag)
		where = append(where, "("+p+" = ANY(tags) OR "+p+" = ANY(imagery) OR "+p+" = ANY(moods))")
	}
	if q.YearFrom > 0 {
		where = append(where, "year >= "+a.add(q.YearFrom))
	}
	if q.YearTo > 0 {
		where = append(where, "year <= "+a.add(q.YearTo))
	}
	if q.Starred {
		where = append(where, "starred")
	}
	from := " FROM lib_songs"
	if len(where) > 0 {
		from += " WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from, a.list...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+libSongColumns+from+` ORDER BY starred DESC, updated_at DESC, id DESC LIMIT `+
		a.add(q.Limit)+` OFFSET `+a.add(q.Offset), a.list...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LibSong{}
	for rows.Next() {
		s, err := scanLibSong(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *s)
	}
	return out, total, rows.Err()
}

func (r *inspirationRepository) GetSong(ctx context.Context, id int64) (*service.LibSong, error) {
	s, err := scanLibSong(r.db.QueryRowContext(ctx, `SELECT `+libSongColumns+` FROM lib_songs WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func libSongArgs(s *service.LibSong) []any {
	allusions, _ := json.Marshal(s.Allusions)
	var year any
	if s.Year != nil {
		year = *s.Year
	}
	return []any{s.Title, s.Artist, s.Lyricist, s.Composer, year, s.Album, s.Lyrics, s.Summary, pq.Array(s.Imagery),
		pq.Array(s.Moods), pq.Array(s.Tags), allusions, s.Notes, s.LinkURL}
}

func (r *inspirationRepository) CreateSong(ctx context.Context, s *service.LibSong) error {
	args := append(libSongArgs(s), s.CreatedBy)
	return r.db.QueryRowContext(ctx, `INSERT INTO lib_songs (title, artist, lyricist, composer, year, album, lyrics, summary, imagery,
       moods, tags, allusions, notes, link_url, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id`, args...).Scan(&s.ID)
}

func (r *inspirationRepository) UpdateSong(ctx context.Context, s *service.LibSong) error {
	args := append(libSongArgs(s), s.ID)
	_, err := r.db.ExecContext(ctx, `UPDATE lib_songs SET title = $1, artist = $2, lyricist = $3, composer = $4, year = $5, album = $6,
       lyrics = $7, summary = $8, imagery = $9, moods = $10, tags = $11, allusions = $12, notes = $13, link_url = $14,
       updated_at = NOW()
WHERE id = $15`, args...)
	return err
}

func (r *inspirationRepository) DeleteSong(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lib_songs WHERE id = $1`, id)
	return err
}

func (r *inspirationRepository) StarSong(ctx context.Context, id int64, starred bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE lib_songs SET starred = $2 WHERE id = $1`, id, starred)
	return err
}

func (r *inspirationRepository) facet(ctx context.Context, query string) ([]service.LibFacet, error) {
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LibFacet{}
	for rows.Next() {
		var f service.LibFacet
		if err := rows.Scan(&f.Name, &f.Count); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *inspirationRepository) SongFacets(ctx context.Context) (*service.LibSongFacets, error) {
	artists, err := r.facet(ctx, `SELECT artist, COUNT(*) FROM lib_songs WHERE artist <> '' GROUP BY artist ORDER BY COUNT(*) DESC, artist LIMIT 300`)
	if err != nil {
		return nil, err
	}
	tags, err := r.facet(ctx, `SELECT t, COUNT(*) FROM lib_songs, unnest(tags || moods) AS t GROUP BY t ORDER BY COUNT(*) DESC, t LIMIT 300`)
	if err != nil {
		return nil, err
	}
	f := &service.LibSongFacets{Artists: artists, Tags: tags, Years: []int{}}
	var lo, hi sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `SELECT MIN(year), MAX(year) FROM lib_songs`).Scan(&lo, &hi); err != nil {
		return nil, err
	}
	if lo.Valid && hi.Valid {
		f.Years = []int{int(lo.Int64), int(hi.Int64)}
	}
	return f, nil
}

func (r *inspirationRepository) AllLyrics(ctx context.Context) ([]service.LibSong, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title, lyrics FROM lib_songs WHERE lyrics <> ''`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LibSong{}
	for rows.Next() {
		var s service.LibSong
		if err := rows.Scan(&s.ID, &s.Title, &s.Lyrics); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---- 诗词灵感 for users: balance and 按画面找 charges ----

func (r *inspirationRepository) Balance(ctx context.Context, userID int64) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrUserNotFound
	}
	return balance, err
}

// ChargeFind takes a run's price from the balance and records it, in one transaction. The balance was
// checked before the run, so the deduction is unconditional (the result is already shown).
func (r *inspirationRepository) ChargeFind(ctx context.Context, userID int64, cost float64, scene, model string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, cost, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrUserNotFound
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lib_find_charges (user_id, cost, scene, model) VALUES ($1, $2, $3, $4)`,
		userID, cost, scene, model); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *inspirationRepository) FindStats(ctx context.Context, since time.Time) (int, float64, error) {
	var n int
	var revenue float64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(cost), 0) FROM lib_find_charges WHERE created_at >= $1`, since).Scan(&n, &revenue)
	return n, revenue, err
}
