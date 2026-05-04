package gmcore_response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func JSON(w http.ResponseWriter, status int, value interface{}) error {
	writeContentType(w, "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}

func JSONPretty(w http.ResponseWriter, status int, value interface{}) error {
	writeContentType(w, "application/json; charset=utf-8")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func HTML(w http.ResponseWriter, status int, value string) error {
	writeContentType(w, "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := io.WriteString(w, value)
	return err
}

func Text(w http.ResponseWriter, status int, value string) error {
	writeContentType(w, "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, err := io.WriteString(w, value)
	return err
}

func Redirect(w http.ResponseWriter, r *http.Request, location string, status int) {
	http.Redirect(w, r, strings.TrimSpace(location), status)
}

func PermanentRedirect(w http.ResponseWriter, r *http.Request, location string) {
	Redirect(w, r, location, http.StatusMovedPermanently)
}

func SeeOther(w http.ResponseWriter, r *http.Request, location string) {
	Redirect(w, r, location, http.StatusSeeOther)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Stream(w http.ResponseWriter, status int, contentType string, reader io.Reader) error {
	writeContentType(w, contentType)
	w.WriteHeader(status)
	_, err := io.Copy(w, reader)
	return err
}

func StreamFunc(w http.ResponseWriter, status int, contentType string, fn func(io.Writer) error) error {
	writeContentType(w, contentType)
	w.WriteHeader(status)
	if fn == nil {
		return nil
	}
	return fn(w)
}

func Download(w http.ResponseWriter, filename string, reader io.Reader, contentType string) error {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", strings.TrimSpace(filename)))
	return Stream(w, http.StatusOK, contentType, reader)
}

func File(w http.ResponseWriter, filename string, contentType string) error {
	handle, err := os.Open(strings.TrimSpace(filename))
	if err != nil {
		if errorsIs(err, fs.ErrNotExist) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return nil
		}
		return err
	}
	defer handle.Close()
	if strings.TrimSpace(contentType) == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	info, err := handle.Stat()
	if err != nil {
		return err
	}
	writeContentType(w, contentType)
	http.ServeContent(w, &http.Request{}, filepath.Base(filename), info.ModTime(), handle)
	return nil
}

func CacheControl(w http.ResponseWriter, maxAge time.Duration, visibility string) {
	parts := []string{firstNonEmpty(visibility, "private")}
	if maxAge <= 0 {
		parts = append(parts, "no-cache", "must-revalidate")
	} else {
		parts = append(parts, "max-age="+strconv.Itoa(int(maxAge.Seconds())))
	}
	w.Header().Set("Cache-Control", strings.Join(parts, ", "))
}

func Cookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	http.SetCookie(w, cookie)
}

func DeleteCookie(w http.ResponseWriter, name string, path string) {
	Cookie(w, &http.Cookie{
		Name:     strings.TrimSpace(name),
		Value:    "",
		Path:     firstNonEmpty(path, "/"),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}

func Header(w http.ResponseWriter, key string, value string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	w.Header().Set(strings.TrimSpace(key), strings.TrimSpace(value))
}

func ETag(w http.ResponseWriter, r *http.Request, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	w.Header().Set("ETag", value)
	if strings.TrimSpace(r.Header.Get("If-None-Match")) == value {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

func LastModified(w http.ResponseWriter, r *http.Request, modifiedAt time.Time) bool {
	if modifiedAt.IsZero() {
		return false
	}
	w.Header().Set("Last-Modified", modifiedAt.UTC().Format(http.TimeFormat))
	ifModifiedSince := strings.TrimSpace(r.Header.Get("If-Modified-Since"))
	if ifModifiedSince == "" {
		return false
	}
	parsed, err := time.Parse(http.TimeFormat, ifModifiedSince)
	if err != nil {
		return false
	}
	if !modifiedAt.UTC().After(parsed.UTC()) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

func PublicCache(w http.ResponseWriter, maxAge time.Duration) {
	CacheControl(w, maxAge, "public")
}

func PrivateCache(w http.ResponseWriter, maxAge time.Duration) {
	CacheControl(w, maxAge, "private")
}

func DownloadFile(w http.ResponseWriter, filename string) error {
	handle, err := os.Open(strings.TrimSpace(filename))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return nil
		}
		return err
	}
	defer handle.Close()
	info, err := handle.Stat()
	if err != nil {
		return err
	}
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(filename)))
	writeContentType(w, contentType)
	http.ServeContent(w, &http.Request{}, filepath.Base(filename), info.ModTime(), handle)
	return nil
}

func Problem(w http.ResponseWriter, status int, title string, detail string) error {
	writeContentType(w, "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"title":  firstNonEmpty(title, http.StatusText(status)),
		"detail": strings.TrimSpace(detail),
	})
}

func Error(w http.ResponseWriter, status int, message string) error {
	payload := map[string]interface{}{
		"status":  status,
		"error":   http.StatusText(status),
		"message": strings.TrimSpace(message),
	}
	return JSON(w, status, payload)
}

func errorsIs(err error, target error) bool { return errors.Is(err, target) }

func writeContentType(w http.ResponseWriter, contentType string) {
	if strings.TrimSpace(w.Header().Get("Content-Type")) == "" {
		w.Header().Set("Content-Type", strings.TrimSpace(contentType))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
