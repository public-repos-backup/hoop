(ns webapp.events.database-schema
  (:require [re-frame.core :as rf]))

(defn- process-schema [schema-data]
  (let [schemas (:schemas schema-data)]
    (reduce (fn [acc schema]
              (let [schema-name (:name schema)
                    tables (reduce (fn [table-acc table]
                                     (assoc table-acc (:name table)
                                            (reduce (fn [col-acc column]
                                                      (assoc col-acc (:name column)
                                                             {(:type column)
                                                              {"nullable" (:nullable column)
                                                               "is_primary_key" (:is_primary_key column)
                                                               "is_foreign_key" (:is_foreign_key column)}}))
                                                    {}
                                                    (:columns table))))
                                   {}
                                   (:tables schema))]
                (assoc acc schema-name tables)))
            {}
            schemas)))

(defn- process-indexes [schema-data]
  (let [schemas (:schemas schema-data)]
    (reduce (fn [acc schema]
              (let [schema-name (:name schema)
                    tables (reduce (fn [table-acc table]
                                     (assoc table-acc (:name table)
                                            (reduce (fn [idx-acc index]
                                                      (assoc idx-acc (:name index)
                                                             (reduce (fn [col-acc column]
                                                                       (assoc col-acc column
                                                                              {"is_unique" (:is_unique index)
                                                                               "is_primary" (:is_primary index)}))
                                                                     {}
                                                                     (:columns index))))
                                                    {}
                                                    (:indexes table))))
                                   {}
                                   (:tables schema))]
                (assoc acc schema-name tables)))
            {}
            schemas)))

(rf/reg-event-fx
 :database-schema->handle-multi-database-schema
 (fn [{:keys [db]} [_ connection]]
   (let [current-connection-data (get-in db [:database-schema :data (:connection-name connection)])
         selected-db (.getItem js/localStorage "selected-database")]
     (if (and selected-db (:databases current-connection-data))
       ;; if there is a selected database, fetch its schema
       {:fx [[:dispatch [:database-schema->get-multi-database-schema
                         connection
                         selected-db
                         (:databases current-connection-data)]]]}
       ;; if not, just fetch the databases
       {:fx [[:dispatch [:database-schema->get-multi-databases connection]]]}))))

(rf/reg-event-fx
 :database-schema->get-multi-databases
 (fn [{:keys [db]} [_ connection]]
   {:db (-> db
            (assoc-in [:database-schema :current-connection] (:connection-name connection))
            (assoc-in [:database-schema :data (:connection-name connection) :status] :loading))
    :fx [[:dispatch [:fetch {:method "GET"
                             :uri (str "/connections/" (:connection-name connection) "/databases")
                             :on-success (fn [response]
                                           (let [selected-db (.getItem js/localStorage "selected-database")]
                                          ;; Se tiver uma database selecionada, já busca seu schema
                                             (when selected-db
                                               (rf/dispatch [:database-schema->get-multi-database-schema
                                                             connection
                                                             selected-db
                                                             (:databases response)]))
                                          ;; Sempre atualiza a lista de databases
                                             (rf/dispatch [:database-schema->set-multi-databases
                                                           connection
                                                           (:databases response)])))}]]]}))

(rf/reg-event-db
 :database-schema->set-multi-databases
 (fn [db [_ connection databases]]
   (assoc-in db [:database-schema :data (:connection-name connection) :databases] databases)))

(rf/reg-event-fx
 :database-schema->get-multi-database-schema
 (fn [{:keys [db]} [_ connection database databases]]
   {:db (-> db
            (assoc-in [:database-schema :data (:connection-name connection) :database-schema-status] :loading)
            (assoc-in [:database-schema :data (:connection-name connection) :databases] databases))
    :fx [[:dispatch [:fetch {:method "GET"
                             :uri (str "/connections/" (:connection-name connection) "/schemas?database=" database)
                             :on-success #(rf/dispatch [:database-schema->set-multi-database-schema
                                                        {:schema-payload %
                                                         :database database
                                                         :databases databases
                                                         :status :success
                                                         :database-schema-status :success
                                                         :connection connection}])}]]]}))

(rf/reg-event-fx
 :database-schema->set-multi-database-schema
 (fn [{:keys [db]} [_ {:keys [schema-payload database databases status database-schema-status connection]}]]
   (let [is-mongodb? (= (:type connection) "mongodb")
         schema {:status status
                 :data (assoc (-> db :database-schema :data)
                              (:connection-name connection)
                              {:status status
                               :database-schema-status database-schema-status
                               :type (:type connection)
                               :raw schema-payload
                               :schema-tree (process-schema schema-payload)
                             ;; only process indexes if it's not a mongodb connection
                               :indexes-tree (when-not is-mongodb?
                                               (process-indexes schema-payload))
                               :current-database database
                               :databases databases})
                 :type (:type connection)
                 :raw schema-payload
                 :schema-tree (process-schema schema-payload)
                 :indexes-tree (when-not is-mongodb?
                                 (process-indexes schema-payload))
                 :current-database database
                 :databases databases}]
     {:db (assoc-in db [:database-schema] schema)})))

(rf/reg-event-fx
 :database-schema->handle-database-schema
 (fn [{:keys [db]} [_ connection]]
   {:db (-> db
            (assoc-in [:database-schema :current-connection] (:connection-name connection))
            (assoc-in [:database-schema :data (:connection-name connection) :status] :loading))
    :fx [[:dispatch [:database-schema->get-database-schema connection]]]}))

(rf/reg-event-fx
 :database-schema->get-database-schema
 (fn [{:keys [db]} [_ connection]]
   {:db (assoc-in db [:database-schema :data (:connection-name connection) :database-schema-status] :loading)
    :fx [[:dispatch [:fetch {:method "GET"
                             :uri (str "/connections/" (:connection-name connection) "/schemas")
                             :on-success #(rf/dispatch [:database-schema->set-database-schema
                                                        {:schema-payload %
                                                         :status :success
                                                         :database-schema-status :success
                                                         :connection connection}])}]]]}))

(rf/reg-event-fx
 :database-schema->set-database-schema
 (fn [{:keys [db]} [_ {:keys [schema-payload status database-schema-status connection]}]]
   (let [schema {:status status
                 :data (assoc (-> db :database-schema :data)
                              (:connection-name connection)
                              {:status status
                               :database-schema-status database-schema-status
                               :type (:type connection)
                               :raw schema-payload
                               :schema-tree (process-schema schema-payload)
                               :indexes-tree (process-indexes schema-payload)})
                 :type (:type connection)
                 :raw schema-payload
                 :schema-tree (process-schema schema-payload)
                 :indexes-tree (process-indexes schema-payload)}]
     {:db (assoc-in db [:database-schema] schema)})))

;; Event unified to handle schema for all databases
(rf/reg-event-fx
 :database-schema->change-database
 (fn [{:keys [db]} [_ connection database]]
   (.setItem js/localStorage "selected-database" database)
   {:fx [[:dispatch [:database-schema->get-multi-database-schema
                     connection
                     database
                     (get-in db [:database-schema :data (:connection-name connection) :databases])]]]}))