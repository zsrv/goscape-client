package metadata

type Metadata struct {
	// Src is the raw per-id model blob; NewModel1 reads vertex/face sections
	// out of it via offsets stored below. Java: Metadata.src (rev-244).
	Src                   []byte
	NumPoints             int // Java: numPoints (was VertexCount)
	NumFaces              int // Java: numFaces (was FaceCount)
	NumT                  int // Java: numT (was TexturedFaceCount)
	VertexOrderOffset     int // Java: vertexOrderOffset (was VertexFlagsOffset)
	VertexXOffset         int // Java: vertexXOffset
	VertexYOffset         int // Java: vertexYOffset
	VertexZOffset         int // Java: vertexZOffset
	VertexLabelOffset     int // Java: vertexLabelOffset (was VertexLabelsOffset)
	FaceIndexOffset       int // Java: faceIndexOffset (was FaceVerticesOffset)
	FaceIndexOrderOffset  int // Java: faceIndexOrderOffset (was FaceOrientationsOffset)
	FaceColourOffset      int // Java: faceColourOffset (was FaceColoursOffset)
	FaceRenderTypeOffset  int // Java: faceRenderTypeOffset (was FaceInfosOffset)
	FacePriorityOffset    int // Java: facePriorityOffset (was FacePrioritiesOffset)
	FaceAlphaOffset       int // Java: faceAlphaOffset (was FaceAlphasOffset)
	FaceLabelOffset       int // Java: faceLabelOffset (was FaceLabelsOffset)
	FaceTextureAxisOffset int // Java: faceTextureAxisOffset
}

func NewMetadata() *Metadata {
	return new(Metadata)
}
