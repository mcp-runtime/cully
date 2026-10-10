package website

import (
	"errors"
)

// global types for any arbitrary value type ment for server.go and linkedin.go

const (
	MaxRequestsPerIPLimit = 20
	MaxHTMLBodyReader     = 1024
	MaxPhotoSize          = 5 << 20
)

// centralised errors, one-time initialisation avoids errors.New(..) overhead on every error case.
var (
	InvalidLinkedinPhotoErr      = errors.New("invalid LinkedIn photo URL")
	PhotoUnavailableErr          = errors.New("photo unavailable")
	PhotoTooLargeErr             = errors.New("photo too large")
	InvalidTestimonialConsentErr = errors.New("Please agree to publication before submitting.")
	InvalidNameLengthErr         = errors.New("Enter a name up to 80 characters.")
	InvalidTestimonialLengthErr  = errors.New("Write a testimonial between 20 and 1,000 characters.")
	InvalidRolesLengthErr        = errors.New("Keep role and workplace within 100 characters each.")
	InvalidLinkedinURLErr        = errors.New("Enter a LinkedIn profile URL starting with https://www.linkedin.com/in/.")
	NonexistentImageErr          = errors.New("Choose a JPG, PNG, or WebP photo.")
	InvalidImageSizeErr          = errors.New("Choose a valid photo no larger than 4096 × 4096 pixels.")
	InvalidReviewStateErr        = errors.New("invalid review state")
	InvalidSubmissionIDErr       = errors.New("invalid submission ID")
)
