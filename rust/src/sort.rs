//! Map-key ordering used by the wire format.
//!
//! `HashMap` iteration order in Rust is implementation-defined and varies per
//! run. The wire format requires keys in a canonical order matching the Go
//! reflection marshaller. Generated `encode_*` functions sort entries via the
//! [`CompareKeys`] trait below before writing them out.

use std::cmp::Ordering;

/// Returns the canonical map-key ordering for `Self`.
///
///   - `String`/`&str` keys compare by raw UTF-8 byte order (matching Go's
///     `sort.Strings`).
///   - Integer keys compare numerically.
///   - Float keys (`f32`, `f64`) sort NaN last; differing NaN payloads compare
///     by raw bit pattern; `-0.0` and `+0.0` compare equal.
///   - `bool` keys order `false` before `true`.
pub trait CompareKeys {
    fn cramberry_cmp(&self, other: &Self) -> Ordering;
}

impl CompareKeys for String {
    fn cramberry_cmp(&self, other: &Self) -> Ordering {
        self.as_bytes().cmp(other.as_bytes())
    }
}

impl CompareKeys for &str {
    fn cramberry_cmp(&self, other: &Self) -> Ordering {
        self.as_bytes().cmp(other.as_bytes())
    }
}

macro_rules! impl_compare_keys_int {
    ($($t:ty),+ $(,)?) => {
        $(
            impl CompareKeys for $t {
                fn cramberry_cmp(&self, other: &Self) -> Ordering {
                    self.cmp(other)
                }
            }
        )+
    };
}
impl_compare_keys_int!(i8, i16, i32, i64, isize, u8, u16, u32, u64, usize);

impl CompareKeys for bool {
    fn cramberry_cmp(&self, other: &Self) -> Ordering {
        match (self, other) {
            (false, true) => Ordering::Less,
            (true, false) => Ordering::Greater,
            _ => Ordering::Equal,
        }
    }
}

impl CompareKeys for f32 {
    fn cramberry_cmp(&self, other: &Self) -> Ordering {
        let a_nan = self.is_nan();
        let b_nan = other.is_nan();
        if a_nan && b_nan {
            return self.to_bits().cmp(&other.to_bits());
        }
        if a_nan {
            return Ordering::Greater;
        }
        if b_nan {
            return Ordering::Less;
        }
        // -0.0 and +0.0 compare equal under partial_cmp.
        self.partial_cmp(other).unwrap_or(Ordering::Equal)
    }
}

impl CompareKeys for f64 {
    fn cramberry_cmp(&self, other: &Self) -> Ordering {
        let a_nan = self.is_nan();
        let b_nan = other.is_nan();
        if a_nan && b_nan {
            return self.to_bits().cmp(&other.to_bits());
        }
        if a_nan {
            return Ordering::Greater;
        }
        if b_nan {
            return Ordering::Less;
        }
        self.partial_cmp(other).unwrap_or(Ordering::Equal)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn string_keys_compare_by_utf8_bytes() {
        // 'a' (0x61) < 'z' (0x7A) < non-ASCII (UTF-8 leading byte >= 0xC0)
        assert_eq!("a".cramberry_cmp(&"z"), Ordering::Less);
        assert_eq!("z".cramberry_cmp(&"\u{4F60}"), Ordering::Less);
    }

    #[test]
    fn float_nan_sorts_last() {
        let nan = f64::NAN;
        assert_eq!(1.0_f64.cramberry_cmp(&nan), Ordering::Less);
        assert_eq!(nan.cramberry_cmp(&1.0_f64), Ordering::Greater);
    }

    #[test]
    fn float_neg_zero_equals_pos_zero() {
        assert_eq!((-0.0_f64).cramberry_cmp(&0.0_f64), Ordering::Equal);
    }

    #[test]
    fn bool_false_before_true() {
        assert_eq!(false.cramberry_cmp(&true), Ordering::Less);
    }
}
