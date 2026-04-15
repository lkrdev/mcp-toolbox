# ==============================================================================
# EXAMPLE 1: BUSINESS PULSE
# ==============================================================================
- dashboard: business_pulse
  title: "Business Pulse"
  layout: newspaper
  preferred_viewer: dashboards-next
  rows:
    - elements: [total_orders, average_order_profit, first_purchasers]
      height: 220
    - elements: [orders_by_day_and_category, sales_by_date]
      height: 400
    - elements: [top_zip_codes, sales_state_map]
      height: 400
    - elements: [sales_by_date_and_category, top_15_brands]
      height: 400
    - elements: [cohort_title]
      height: 50
    - elements: [layer_cake_cohort]
      height: 400
    - elements: [customer_cohort]
      height: 400

  filters:
  - name: date
    title: "Date"
    type: date_filter
    default_value: Last 90 Days

  - name: state
    title: 'State / Region'
    type: field_filter
    explore: users
    field: users.state

  elements:
  - name: total_orders
    type: single_value
    explore: orders
    measures: [orders.count]
    listen:
      date: orders.created_date
      state: users.state
    font_size: medium

---
# ==============================================================================
# EXAMPLE 2: BRAND LOOKUP
# ==============================================================================
- dashboard: brand_lookup
  title: Brand Lookup
  layout: newspaper
  preferred_viewer: dashboards-next
  query_timezone: user_timezone
  embed_style:
    background_color: "#f6f8fa"
    show_title: true
    title_color: "#3a4245"
    show_filters_bar: true
    tile_text_color: "#3a4245"
    text_tile_text_color: "#556d7a"
  elements:
  - title: Total Orders
    name: Total Orders
    model: thelook
    explore: order_items
    type: single_value
    fields: [order_items.order_count]
    filters: {}
    sorts: [order_items.order_count desc]
    limit: 500
    query_timezone: America/Los_Angeles
    font_size: medium
    text_color: black
    listen:
      Brand Name: products.brand
      Date: order_items.created_date
      State: users.state
    row: 2
    col: 8
    width: 4
    height: 3

---
# ==============================================================================
# EXAMPLE 3: CUSTOMER LOOKUP
# ==============================================================================
- dashboard: customer_lookup
  title: Customer Lookup
  layout: newspaper
  description: ''
  preferred_slug: MDDG8M9Lvb1S2zq5UuhUND
  embed_style:
    background_color: "#f6f8fa"
    show_title: true
    title_color: "#3a4245"
    show_filters_bar: true
    tile_text_color: "#3a4245"
    text_tile_text_color: ''
  elements:
  - title: User Info
    name: User Info
    model: thelook
    explore: order_items
    type: looker_single_record
    fields: [users.id, users.email, users.name, users.traffic_source, users.created_month,
      users.age, users.gender, users.city, users.state]
    filters:
      order_items.created_date: 99 years
      users.id: ''
    sorts: [users.created_month desc]
    limit: 1
    column_limit: 50
    query_timezone: America/Los_Angeles
    show_view_names: false
    show_null_labels: false
    show_row_numbers: true
    hidden_fields: []
    y_axes: []
    defaults_version: 1
    listen:
      Email: users.email
    row: 0
    col: 0
    width: 7
    height: 6

---
# ==============================================================================
# EXAMPLE 4: WEB ANALYTICS OVERVIEW
# ==============================================================================
- dashboard: web_analytics_overview
  title: Web Analytics Overview
  layout: newspaper
  preferred_viewer: dashboards-next
  query_timezone: user_timezone
  preferred_slug: 2VgEQ4QWmU1qoZFiFsSd3K
  embed_style:
    background_color: ''
    show_title: true
    title_color: "#131414"
    show_filters_bar: true
    tile_text_color: "#070808"
    text_tile_text_color: "#0d0d0c"
  elements:
  - title: Total Visitors
    name: Total Visitors
    model: thelook
    explore: events
    type: single_value
    fields: [events.unique_visitors, events.event_week]
    filters:
      events.event_date: 2 weeks ago for 2 weeks
    sorts: [events.event_week desc]
    limit: 500
    column_limit: 50
    dynamic_fields: [{table_calculation: change, label: Change, expression: "${events.unique_visitors}-offset(${events.unique_visitors},1)"}]
    query_timezone: America/Los_Angeles
    font_size: medium
    value_format: ''
    text_color: black
    colors: ["#1f78b4", "#a6cee3", "#33a02c", "#b2df8a", "#e31a1c", "#fb9a99", "#ff7f00",
      "#fdbf6f", "#6a3d9a", "#cab2d6", "#b15928", "#edbc0e"]
    show_single_value_title: true
    show_comparison: true
    comparison_type: change
    comparison_reverse_colors: false
    show_comparison_label: true
    comparison_label: Weekly Change
    single_value_title: Visitors Past Week
    note_state: collapsed
    note_display: below
    note_text: ''
    listen:
      Browser: events.browser
      Traffic Source: users.traffic_source
    row: 0
    col: 0
    width: 6
    height: 3
