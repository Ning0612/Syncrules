package gdrive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/Ning0612/Syncrules/internal/domain"
)

const (
	// MimeTypeFolder is the MIME type for Google Drive folders
	MimeTypeFolder = "application/vnd.google-apps.folder"
	// PageSize is the number of files to fetch per request
	PageSize = 100
)

// Adapter implements the adapter.Adapter interface for Google Drive
type Adapter struct {
	service *drive.Service
	root    string   // Root folder path in Drive (e.g., "/SyncRules/backup")
	rootID  string   // Cached root folder ID
	cache   *idCache // Cache for path -> ID mapping
}

// idCache caches folder ID lookups with thread-safe access
type idCache struct {
	mu    sync.RWMutex
	paths map[string]string // path -> folder ID
}

func newIDCache() *idCache {
	return &idCache{
		paths: make(map[string]string),
	}
}

func (c *idCache) get(path string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.paths[path]
	return id, ok
}

func (c *idCache) set(path, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paths[path] = id
}

func (c *idCache) delete(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.paths, path)
}

// New creates a new Google Drive adapter
func New(ctx context.Context, clientID, clientSecret, tokenPath, root string) (*Adapter, error) {
	auth := NewAuthenticator(clientID, clientSecret, tokenPath)

	token, err := auth.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	// Create authenticated client
	client := auth.Config().Client(ctx, token)

	// Create Drive service
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	adapter := &Adapter{
		service: service,
		root:    normalizeRoot(root),
		cache:   newIDCache(),
	}

	// Resolve root folder ID
	rootID, err := adapter.resolveOrCreatePath(ctx, adapter.root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root folder: %w", err)
	}
	adapter.rootID = rootID
	adapter.cache.set(adapter.root, rootID)

	return adapter, nil
}

// NewWithToken creates a new adapter with an existing token
func NewWithToken(ctx context.Context, token *oauth2.Token, oauthConfig *oauth2.Config, root string) (*Adapter, error) {
	client := oauthConfig.Client(ctx, token)

	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	adapter := &Adapter{
		service: service,
		root:    normalizeRoot(root),
		cache:   newIDCache(),
	}

	rootID, err := adapter.resolveOrCreatePath(ctx, adapter.root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root folder: %w", err)
	}
	adapter.rootID = rootID
	adapter.cache.set(adapter.root, rootID)

	return adapter, nil
}

// normalizeRoot normalizes the root path
func normalizeRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" || root == "/" {
		return ""
	}
	// Ensure leading slash, no trailing slash
	if !strings.HasPrefix(root, "/") {
		root = "/" + root
	}
	return strings.TrimSuffix(root, "/")
}

// List returns all files and directories under the given path
func (a *Adapter) List(ctx context.Context, relPath string) ([]domain.FileInfo, error) {
	fullPath, err := a.joinPath(relPath)
	if err != nil {
		return nil, err
	}
	// Use getFileID instead of getOrCreateFolderID to avoid implicit folder creation
	folderID, err := a.getFileID(ctx, fullPath)
	if err != nil {
		return nil, err
	}

	var result []domain.FileInfo
	pageToken := ""

	for {
		query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
		call := a.service.Files.List().
			Q(query).
			PageSize(PageSize).
			Fields("nextPageToken, files(id, name, mimeType, size, modifiedTime, md5Checksum)")

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		fileList, err := call.Context(ctx).Do()
		if err != nil {
			return nil, a.mapError(err)
		}

		for _, f := range fileList.Files {
			info := a.fileInfoFromDrive(relPath, f)
			result = append(result, info)
		}

		pageToken = fileList.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return result, nil
}

// Read opens a file for reading
func (a *Adapter) Read(ctx context.Context, relPath string) (io.ReadCloser, error) {
	fullPath, err := a.joinPath(relPath)
	if err != nil {
		return nil, err
	}
	fileID, err := a.getFileID(ctx, fullPath)
	if err != nil {
		return nil, err
	}

	resp, err := a.service.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return nil, a.mapError(err)
	}

	return resp.Body, nil
}

// Write creates or overwrites a file
func (a *Adapter) Write(ctx context.Context, relPath string, r io.Reader) error {
	fullPath, err := a.joinPath(relPath)
	if err != nil {
		return err
	}
	dirPath := path.Dir(fullPath)
	fileName := path.Base(fullPath)

	// Check if file already exists
	existingID, err := a.getFileID(ctx, fullPath)
	if err == nil {
		// File exists, update it
		file := &drive.File{
			Name: fileName,
		}
		_, updateErr := a.service.Files.Update(existingID, file).
			Context(ctx).
			Media(r).
			Do()
		return a.mapError(updateErr)
	}

	// Only create new file if error is ErrNotFound
	// Other errors (permission, network, etc.) should be propagated
	if err != domain.ErrNotFound {
		return err
	}

	// Ensure parent directory exists
	parentID, err := a.getOrCreateFolderID(ctx, dirPath)
	if err != nil {
		return err
	}

	// File doesn't exist, create it
	file := &drive.File{
		Name:    fileName,
		Parents: []string{parentID},
	}
	_, err = a.service.Files.Create(file).
		Context(ctx).
		Media(r).
		Do()
	return a.mapError(err)
}

// Delete removes a file or empty directory
func (a *Adapter) Delete(ctx context.Context, relPath string) error {
	fullPath, err := a.joinPath(relPath)
	if err != nil {
		return err
	}
	fileID, err := a.getFileID(ctx, fullPath)
	if err != nil {
		return err
	}

	err = a.service.Files.Delete(fileID).Context(ctx).Do()
	if err != nil {
		return a.mapError(err)
	}

	// Clear from cache
	a.cache.delete(fullPath)
	return nil
}

// Stat returns metadata for a single path
func (a *Adapter) Stat(ctx context.Context, relPath string) (domain.FileInfo, error) {
	fullPath, err := a.joinPath(relPath)
	if err != nil {
		return domain.FileInfo{}, err
	}
	fileID, err := a.getFileID(ctx, fullPath)
	if err != nil {
		return domain.FileInfo{}, err
	}

	file, err := a.service.Files.Get(fileID).
		Fields("id, name, mimeType, size, modifiedTime, md5Checksum").
		Context(ctx).Do()
	if err != nil {
		return domain.FileInfo{}, a.mapError(err)
	}

	return a.fileInfoFromDrive(path.Dir(relPath), file), nil
}

// StatWithChecksum returns metadata including checksum for a file
func (a *Adapter) StatWithChecksum(ctx context.Context, relPath string) (domain.FileInfo, error) {
	// Google Drive provides MD5 checksum in file metadata
	return a.Stat(ctx, relPath)
}

// Mkdir creates a directory and any necessary parents
func (a *Adapter) Mkdir(ctx context.Context, relPath string) error {
	fullPath, err := a.joinPath(relPath)
	if err != nil {
		return err
	}
	_, err = a.getOrCreateFolderID(ctx, fullPath)
	return err
}

// Exists checks if a path exists
func (a *Adapter) Exists(ctx context.Context, relPath string) (bool, error) {
	fullPath, err := a.joinPath(relPath)
	if err != nil {
		return false, err
	}
	_, err = a.getFileID(ctx, fullPath)
	if err == domain.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Close releases any resources
func (a *Adapter) Close() error {
	return nil
}

// Root returns the root path of this adapter
func (a *Adapter) Root() string {
	return a.root
}

// joinPath joins relative path with root and validates against path traversal
func (a *Adapter) joinPath(relPath string) (string, error) {
	if relPath == "" || relPath == "." {
		return a.root, nil
	}

	// Clean the path to handle .. and .
	cleanPath := path.Clean(relPath)

	// Reject absolute paths
	if path.IsAbs(cleanPath) {
		return "", domain.ErrPermissionDenied
	}

	// Check for path traversal attempt
	if strings.HasPrefix(cleanPath, "..") {
		return "", domain.ErrPermissionDenied
	}

	fullPath := path.Join(a.root, cleanPath)

	// Verify the result is still under root (handles edge cases)
	if a.root != "" && !strings.HasPrefix(fullPath, a.root) {
		return "", domain.ErrPermissionDenied
	}

	return fullPath, nil
}

// escapeQueryString escapes special characters in Drive query strings
func escapeQueryString(s string) string {
	// Escape backslash first, then single quote
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// getFileID returns the ID of a file or folder at the given path
func (a *Adapter) getFileID(ctx context.Context, fullPath string) (string, error) {
	// Check cache first
	if id, ok := a.cache.get(fullPath); ok {
		return id, nil
	}

	// Empty path means root of Drive
	if fullPath == "" {
		return "root", nil
	}

	// Walk the path from root
	parts := strings.Split(strings.TrimPrefix(fullPath, "/"), "/")
	currentID := "root"

	for i, part := range parts {
		if part == "" {
			continue
		}

		// Escape single quotes to prevent query injection
		escapedPart := escapeQueryString(part)
		query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", escapedPart, currentID)
		fileList, err := a.service.Files.List().
			Q(query).
			PageSize(1).
			Fields("files(id, mimeType)").
			Context(ctx).Do()
		if err != nil {
			return "", a.mapError(err)
		}

		if len(fileList.Files) == 0 {
			return "", domain.ErrNotFound
		}

		currentID = fileList.Files[0].Id

		// Cache intermediate paths
		partialPath := "/" + strings.Join(parts[:i+1], "/")
		a.cache.set(partialPath, currentID)
	}

	return currentID, nil
}

// getOrCreateFolderID returns the ID of a folder, creating it if necessary
func (a *Adapter) getOrCreateFolderID(ctx context.Context, fullPath string) (string, error) {
	if fullPath == "" {
		return "root", nil
	}

	// Check cache
	if id, ok := a.cache.get(fullPath); ok {
		return id, nil
	}

	parts := strings.Split(strings.TrimPrefix(fullPath, "/"), "/")
	currentID := "root"

	for i, part := range parts {
		if part == "" {
			continue
		}

		partialPath := "/" + strings.Join(parts[:i+1], "/")

		// Check cache for this partial path
		if id, ok := a.cache.get(partialPath); ok {
			currentID = id
			continue
		}

		// Look for existing folder with escaped query
		escapedPart := escapeQueryString(part)
		query := fmt.Sprintf("name = '%s' and '%s' in parents and mimeType = '%s' and trashed = false",
			escapedPart, currentID, MimeTypeFolder)
		fileList, err := a.service.Files.List().
			Q(query).
			PageSize(1).
			Fields("files(id)").
			Context(ctx).Do()
		if err != nil {
			return "", a.mapError(err)
		}

		if len(fileList.Files) > 0 {
			currentID = fileList.Files[0].Id
		} else {
			// Create folder
			folder := &drive.File{
				Name:     part,
				MimeType: MimeTypeFolder,
				Parents:  []string{currentID},
			}
			created, err := a.service.Files.Create(folder).
				Fields("id").
				Context(ctx).Do()
			if err != nil {
				return "", a.mapError(err)
			}
			currentID = created.Id
		}

		// Cache
		a.cache.set(partialPath, currentID)
	}

	return currentID, nil
}

// resolveOrCreatePath resolves a path and creates folders if needed
func (a *Adapter) resolveOrCreatePath(ctx context.Context, fullPath string) (string, error) {
	return a.getOrCreateFolderID(ctx, fullPath)
}

// fileInfoFromDrive converts a Drive file to domain.FileInfo
func (a *Adapter) fileInfoFromDrive(parentPath string, file *drive.File) domain.FileInfo {
	fileType := domain.FileTypeRegular
	if file.MimeType == MimeTypeFolder {
		fileType = domain.FileTypeDirectory
	}

	modTime := time.Time{}
	if file.ModifiedTime != "" {
		modTime, _ = time.Parse(time.RFC3339, file.ModifiedTime)
	}

	filePath := file.Name
	if parentPath != "" && parentPath != "." {
		filePath = path.Join(parentPath, file.Name)
	}

	return domain.FileInfo{
		Path:     filePath,
		Type:     fileType,
		Size:     file.Size,
		ModTime:  modTime,
		Checksum: file.Md5Checksum, // Drive provides MD5
	}
}

// mapError converts Google API errors to domain errors
func (a *Adapter) mapError(err error) error {
	if err == nil {
		return nil
	}

	// Use Google API error types for more reliable error detection
	var apiErr *googleapi.Error
	if ok := errors.As(err, &apiErr); ok {
		switch apiErr.Code {
		case 404:
			return domain.ErrNotFound
		case 403:
			return domain.ErrPermissionDenied
		case 409:
			return domain.ErrAlreadyExists
		case 429:
			// Rate limit - return original error with context
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Fallback to string matching for non-googleapi errors
	errStr := err.Error()
	if strings.Contains(errStr, "notFound") {
		return domain.ErrNotFound
	}

	return err
}

// Compile-time interface check
var _ io.Closer = (*Adapter)(nil)

// BufferedReader wraps content in a bytes.Reader for re-reading
type BufferedReader struct {
	*bytes.Reader
}

// Close implements io.Closer
func (br *BufferedReader) Close() error {
	return nil
}
