package api_test

import (
	"context"
	"database/sql"

	"db-design-project/internal/db"
)

// mockQuerier implements db.Querier. Set any Fn field to override the default
// zero-value return. Methods that are not set return zero values with nil error,
// so each test only specifies what it cares about.
type mockQuerier struct {
	getUserByEmailFn        func(ctx context.Context, email string) (db.User, error)
	listAvailableProductsFn func(ctx context.Context, arg db.ListAvailableProductsParams) ([]db.Product, error)
	getProductByIDFn        func(ctx context.Context, id int32) (db.Product, error)
	createOrderFn           func(ctx context.Context, arg db.CreateOrderParams) (db.Order, error)
	listOrdersByUserFn      func(ctx context.Context, userID int32) ([]db.Order, error)
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return db.User{}, sql.ErrNoRows
}

func (m *mockQuerier) ListAvailableProducts(ctx context.Context, arg db.ListAvailableProductsParams) ([]db.Product, error) {
	if m.listAvailableProductsFn != nil {
		return m.listAvailableProductsFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) GetProductByID(ctx context.Context, id int32) (db.Product, error) {
	if m.getProductByIDFn != nil {
		return m.getProductByIDFn(ctx, id)
	}
	return db.Product{}, sql.ErrNoRows
}

func (m *mockQuerier) CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.Order, error) {
	if m.createOrderFn != nil {
		return m.createOrderFn(ctx, arg)
	}
	return db.Order{}, nil
}

func (m *mockQuerier) ListOrdersByUser(ctx context.Context, userID int32) ([]db.Order, error) {
	if m.listOrdersByUserFn != nil {
		return m.listOrdersByUserFn(ctx, userID)
	}
	return nil, nil
}

// Remaining interface methods — not exercised by current handlers.
func (m *mockQuerier) AddOrderItem(ctx context.Context, arg db.AddOrderItemParams) (db.OrderItem, error) {
	return db.OrderItem{}, nil
}
func (m *mockQuerier) CalculateOrderTotal(ctx context.Context, orderID int32) (string, error) {
	return "0", nil
}
func (m *mockQuerier) CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error) {
	return db.Product{}, nil
}
func (m *mockQuerier) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	return db.User{}, nil
}
func (m *mockQuerier) GetOrderByID(ctx context.Context, id int32) (db.Order, error) {
	return db.Order{}, nil
}
func (m *mockQuerier) GetProductsByCategory(ctx context.Context, id int32) ([]db.Product, error) {
	return nil, nil
}
func (m *mockQuerier) GetUserByID(ctx context.Context, id int32) (db.User, error) {
	return db.User{}, nil
}
func (m *mockQuerier) ListCategories(ctx context.Context) ([]db.Category, error) {
	return nil, nil
}
func (m *mockQuerier) ListOrderItems(ctx context.Context, orderID int32) ([]db.ListOrderItemsRow, error) {
	return nil, nil
}
func (m *mockQuerier) SearchProductsByName(ctx context.Context, dollar_1 sql.NullString) ([]db.Product, error) {
	return nil, nil
}
func (m *mockQuerier) UpdateOrderStatus(ctx context.Context, arg db.UpdateOrderStatusParams) error {
	return nil
}
