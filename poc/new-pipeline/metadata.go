package main

type MetadataRegistry struct {
	r []*Metadata
}

type Metadata struct {
	m []map[string]string
}

func (m Metadata) Copy() *Metadata {
	newMetadata := make([]map[string]string, 0)

	for _, metadata := range m.m {
		newMap := make(map[string]string)
		for key, value := range metadata {
			metadata[key] = value
		}

		newMetadata = append(newMetadata, newMap)
	}

	return &Metadata{
		m: newMetadata,
	}
}
