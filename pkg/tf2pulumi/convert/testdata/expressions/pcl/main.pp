output nullOut {
    value = null
}
output numberOut {
    value = 0
}
output boolOut {
    value = true
}
output stringOut {
    value = "hello world"
}
output tupleOut {
    value = [1, 2, 3]
}
output strObjectOut {
    value = {
        hello: "hallo"
        goodbye: "ha det"
    }
}
    aKey = "hello"
    aValue = -1
output complexObjectOut {
    value = {
        a_tuple: ["a", "b", "c"]
        an_object: {
            literal_key: 1
            another_literal_key = 2
            "yet_another_literal_key": aValue
            # This doesn't translate correctly
            # (local.a_key) = 4
        }
        ambiguous_for: {
            "for" = 1
        }
    }
}
output quotedTemplate {
    value = "The key is ${aKey}"
}
output heredoc {
    value = <<END
This is also a template.
So we can output the key again ${aKey}
END
}
output forTuple {
    value = [for key, value in ["a", "b"] : "${key}:${value}:${aValue}" if key != 0]
}
output forTupleValueOnly {
    value = [for value in ["a", "b"] : "${value}:${aValue}"]
}
output forObject {
    value = {for key, value in ["a", "b"] : key => "${value}:${aValue}" if key != 0}
}