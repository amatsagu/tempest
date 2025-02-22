1. Funkcje/Metody nie mogą przekroczyć 60 linijek kodu lub 15 punktów gocyclo (każdy if/else/for/switch daje +1).
2. Każdy kluczowy komponent jak Rest czy Client są interfejsem i wtedy implementują bazową klase.
3. Nie stosujemy żadnej wersji rekursji.
4. Przy definiowaniu struktur Discord API - każda z nich musi mieć dołączony link do dokumentacji.
5. Wszelkie zwracane błędy powinny być z małej litery - wyjątkiem są specjalnie słowa jak "ID", "TOKEN".
6. Każda funkcja/metoda powinna idealnie być wystarczająco prosta, żeby uzyskać "inline optimization" z llvm kompilacji. To się łączy z pierwszym punktem (wymagania gocyclo). Te funkcje gdzie jest to niemożlwe - powinny mieć dołączony komentarz wspominający o tym.
7. Każde miejsce gdzie może wystąpić problem powinno zostać poprawnie rozwiązane. Zakazane jest ignorowanie potencjalnych error value.
8. W nawiązaniu do punktu 2 -> Każdy kluczowy komponent powinien dodatkowo zawierać "interface implementation assertion".