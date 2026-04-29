package wire

// MaxFieldNumber is the maximum allowed field number.
// Matches the limit enforced by the schema validator.
const MaxFieldNumber = 1<<29 - 1 // ~536 million
