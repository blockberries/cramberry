#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_scalar_types_to_json() {
        let msg = ScalarTypes {
            bool_val: true,
            int8_val: -42,
            int16_val: -1000,
            int32_val: -100000,
            int64_val: -9223372036854775807,
            uint8_val: 255,
            uint16_val: 65535,
            uint32_val: 4294967295,
            uint64_val: 18446744073709551615,
            float32_val: 3.14159,
            float64_val: 2.718281828459045,
            string_val: "hello, world!".to_string(),
            bytes_val: vec![0xde, 0xad, 0xbe, 0xef],
        };

        let json = to_json_scalar_types(&msg).expect("to_json failed");
        println!("JSON output: {}", json);

        // Verify round-trip
        let decoded = from_json_scalar_types(&json).expect("from_json failed");
        let json2 = to_json_scalar_types(&decoded).expect("to_json failed");

        assert_eq!(json, json2, "Round-trip failed");
    }

    #[test]
    fn test_enum_json_encoding() {
        let msg = EnumTest {
            status: Status::Active,
            statuses: vec![Status::Active, Status::Pending, Status::Inactive],
        };

        let json = to_json_enum_test(&msg).expect("to_json failed");
        println!("Enum JSON: {}", json);

        // Verify enums are encoded as string names
        assert!(json.contains("\"status\":\"ACTIVE\""));
        assert!(json.contains("[\"ACTIVE\",\"PENDING\",\"INACTIVE\"]"));

        // Round-trip
        let decoded = from_json_enum_test(&json).expect("from_json failed");
        assert_eq!(decoded.status, msg.status);
    }

    #[test]
    fn test_all_zero_values() {
        let msg = AllZeroValues {
            zero_int: 0,
            zero_string: "".to_string(),
            zero_bool: false,
            empty_array: vec![],
        };

        let json = to_json_all_zero_values(&msg).expect("to_json failed");
        println!("Zero values JSON: {}", json);

        // Verify all fields are included
        let expected = r#"{"zero_int":"0","zero_string":"","zero_bool":false,"empty_array":[]}"#;
        assert_eq!(json, expected);
    }

    #[test]
    fn test_unknown_fields_rejected() {
        let json = r#"{"zero_int":"0","unknown_field":"should_fail"}"#;
        let result = from_json_all_zero_values(json);
        assert!(result.is_err(), "Should reject unknown fields");
        println!("Unknown field correctly rejected: {:?}", result);
    }
}
