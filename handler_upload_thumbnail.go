package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "only jpeg and png files are supported", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println(err)
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("video %s not found", videoID), err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Unable to get video from db", err)
		return
	}

	if video.UserID != userID {
		log.Println("unauthorized to edit video thumbnail")
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	fileExtension, err := mime.ExtensionsByType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing mime type", err)
		return
	}

	fileName := fmt.Sprintf("%s.%s", videoID, fileExtension)
	filePath := filepath.Join(cfg.assetsRoot, fileName)

	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create file %s", filePath), err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:8091/assets/%s.%s", videoID, fileExtension)

	io.Copy(newFile, file)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println(err)
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("video %s not found", video.ID), err)
			return
		}
	}

	respondWithJSON(w, http.StatusOK, video)
}
