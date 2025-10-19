@startuml

entity Article {
    + ID: String        // 例如 1750603622.A.960
    + Title: String
    + Board: String     // 例如 "Gossiping"
    + Index: Integer
    + Author: String
    + Date: String      // MM/DD
    + PushCount: Integer
    + Mark: String
    + Url: String
    + Extra: JSONB
}

entity BottomArticle {
    + ID: String        // 例如 1737398137.A.644
    + Title: String
    + Board: String     // 例如 "Gossiping"
    + Index: Integer
    + Author: String
    + Date: String
    + PushCount: Integer
    + Mark: String
    + Url: String
    + Extra: JSONB
}

@enduml