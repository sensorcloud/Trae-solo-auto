import React, { useEffect, useState, useRef } from 'react';
import * as echarts from 'echarts';

interface Order {
  id: string;
  type: 'buy' | 'sell';
  price: number;
  volume: number;
  status: 'pending' | 'partial' | 'filled' | 'cancelled';
  createTime: string;
  filledVolume: number;
  counterparty?: string;
}

interface OrderBookEntry {
  price: number;
  buyVolume: number;
  sellVolume: number;
}

interface Trade {
  id: string;
  price: number;
  volume: number;
  time: string;
  buyer: string;
  seller: string;
}

const TradingCenter: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'spot' | 'orderbook' | 'history'>('spot');
  const [orders, setOrders] = useState<Order[]>([]);
  const [orderBook, setOrderBook] = useState<OrderBookEntry[]>([]);
  const [trades, setTrades] = useState<Trade[]>([]);
  const [loading, setLoading] = useState(true);
  const [orderType, setOrderType] = useState<'buy' | 'sell'>('buy');
  const [orderPrice, setOrderPrice] = useState('0.45');
  const [orderVolume, setOrderVolume] = useState('1000');

  const priceChartRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchOrders();
    fetchOrderBook();
    fetchTrades();
    setLoading(false);
  }, []);

  useEffect(() => {
    if (priceChartRef.current) {
      initPriceChart();
    }
  }, []);

  const fetchOrders = () => {
    const mockOrders: Order[] = [
      {
        id: 'ORD001',
        type: 'buy',
        price: 0.42,
        volume: 5000,
        status: 'filled',
        createTime: '2024-01-15 09:30:00',
        filledVolume: 5000,
        counterparty: '绿电供应商A',
      },
      {
        id: 'ORD002',
        type: 'sell',
        price: 0.48,
        volume: 3000,
        status: 'partial',
        createTime: '2024-01-15 10:15:00',
        filledVolume: 1500,
        counterparty: '数据中心B',
      },
      {
        id: 'ORD003',
        type: 'buy',
        price: 0.44,
        volume: 8000,
        status: 'pending',
        createTime: '2024-01-15 11:00:00',
        filledVolume: 0,
      },
      {
        id: 'ORD004',
        type: 'sell',
        price: 0.50,
        volume: 2000,
        status: 'cancelled',
        createTime: '2024-01-15 11:30:00',
        filledVolume: 0,
      },
    ];
    setOrders(mockOrders);
  };

  const fetchOrderBook = () => {
    const mockOrderBook: OrderBookEntry[] = [
      { price: 0.40, buyVolume: 15000, sellVolume: 0 },
      { price: 0.41, buyVolume: 12000, sellVolume: 0 },
      { price: 0.42, buyVolume: 8000, sellVolume: 0 },
      { price: 0.43, buyVolume: 5000, sellVolume: 0 },
      { price: 0.44, buyVolume: 3000, sellVolume: 2000 },
      { price: 0.45, buyVolume: 1000, sellVolume: 4000 },
      { price: 0.46, buyVolume: 0, sellVolume: 6000 },
      { price: 0.47, buyVolume: 0, sellVolume: 8000 },
      { price: 0.48, buyVolume: 0, sellVolume: 10000 },
      { price: 0.49, buyVolume: 0, sellVolume: 12000 },
    ];
    setOrderBook(mockOrderBook);
  };

  const fetchTrades = () => {
    const mockTrades: Trade[] = [
      { id: 'TRD001', price: 0.44, volume: 2000, time: '11:45:30', buyer: '我方', seller: '绿电供应商A' },
      { id: 'TRD002', price: 0.43, volume: 1500, time: '11:30:15', buyer: '我方', seller: '风电场B' },
      { id: 'TRD003', price: 0.45, volume: 3000, time: '11:15:00', buyer: '数据中心C', seller: '我方' },
      { id: 'TRD004', price: 0.42, volume: 5000, time: '10:45:22', buyer: '我方', seller: '光伏电站D' },
      { id: 'TRD005', price: 0.46, volume: 1000, time: '10:30:45', buyer: '工厂E', seller: '我方' },
    ];
    setTrades(mockTrades);
  };

  const initPriceChart = () => {
    const chart = echarts.init(priceChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const times = [];
    const prices = [];
    const now = new Date();
    for (let i = 60; i >= 0; i--) {
      const time = new Date(now.getTime() - i * 60000);
      times.push(time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }));
      prices.push(0.42 + Math.random() * 0.08);
    }

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => `${params[0].axisValue}<br/>价格: ¥${params[0].value.toFixed(3)}/kWh`,
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        top: '10%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: times,
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b', fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        name: '¥/kWh',
        min: 0.4,
        max: 0.52,
        nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
        splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#f1f5f9' } },
      },
      series: [
        {
          name: '价格',
          type: 'line',
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(16, 185, 129, 0.3)' },
              { offset: 1, color: 'rgba(16, 185, 129, 0.05)' },
            ]),
          },
          data: prices,
        },
      ],
    };

    chart.setOption(option);

    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
      chart.dispose();
    };
  };

  const handleSubmitOrder = () => {
    const newOrder: Order = {
      id: `ORD${String(orders.length + 1).padStart(3, '0')}`,
      type: orderType,
      price: parseFloat(orderPrice),
      volume: parseInt(orderVolume),
      status: 'pending',
      createTime: new Date().toLocaleString('zh-CN'),
      filledVolume: 0,
    };
    setOrders([newOrder, ...orders]);
  };

  const getStatusBadge = (status: Order['status']) => {
    const badges = {
      pending: { text: '待成交', class: 'pending' },
      partial: { text: '部分成交', class: 'partial' },
      filled: { text: '已成交', class: 'filled' },
      cancelled: { text: '已取消', class: 'cancelled' },
    };
    return badges[status];
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="trading-center">
      <div className="page-header">
        <h1 className="page-title">交易中心</h1>
      </div>

      <div className="tabs">
        <button
          className={`tab ${activeTab === 'spot' ? 'active' : ''}`}
          onClick={() => setActiveTab('spot')}
        >
          现货交易
        </button>
        <button
          className={`tab ${activeTab === 'orderbook' ? 'active' : ''}`}
          onClick={() => setActiveTab('orderbook')}
        >
          订单簿
        </button>
        <button
          className={`tab ${activeTab === 'history' ? 'active' : ''}`}
          onClick={() => setActiveTab('history')}
        >
          交易历史
        </button>
      </div>

      {activeTab === 'spot' && (
        <div className="spot-trading">
          <div className="trading-layout">
            <div className="card order-form-card">
              <div className="card-header">
                <h3>下单</h3>
              </div>
              <div className="order-type-tabs">
                <button
                  className={`type-tab ${orderType === 'buy' ? 'buy active' : ''}`}
                  onClick={() => setOrderType('buy')}
                >
                  买入
                </button>
                <button
                  className={`type-tab ${orderType === 'sell' ? 'sell active' : ''}`}
                  onClick={() => setOrderType('sell')}
                >
                  卖出
                </button>
              </div>
              <div className="form-group">
                <label>价格 (¥/kWh)</label>
                <input
                  type="number"
                  step="0.01"
                  className="form-input"
                  value={orderPrice}
                  onChange={e => setOrderPrice(e.target.value)}
                />
              </div>
              <div className="form-group">
                <label>数量 (kWh)</label>
                <input
                  type="number"
                  className="form-input"
                  value={orderVolume}
                  onChange={e => setOrderVolume(e.target.value)}
                />
              </div>
              <div className="order-summary">
                <div className="summary-row">
                  <span>订单金额</span>
                  <span>¥{(parseFloat(orderPrice) * parseInt(orderVolume) || 0).toFixed(2)}</span>
                </div>
              </div>
              <button
                className={`btn submit-btn ${orderType}`}
                onClick={handleSubmitOrder}
              >
                {orderType === 'buy' ? '买入' : '卖出'}
              </button>
            </div>

            <div className="card price-chart-card">
              <div className="card-header">
                <h3>实时价格走势</h3>
              </div>
              <div ref={priceChartRef} className="chart-container" />
            </div>
          </div>

          <div className="card my-orders-card">
            <div className="card-header">
              <h3>我的订单</h3>
            </div>
            <div className="orders-table">
              <div className="table-header">
                <div className="table-cell">订单号</div>
                <div className="table-cell">类型</div>
                <div className="table-cell">价格</div>
                <div className="table-cell">数量</div>
                <div className="table-cell">已成交</div>
                <div className="table-cell">状态</div>
                <div className="table-cell">创建时间</div>
              </div>
              {orders.map(order => (
                <div key={order.id} className="table-row">
                  <div className="table-cell">{order.id}</div>
                  <div className="table-cell">
                    <span className={`type-badge ${order.type}`}>
                      {order.type === 'buy' ? '买入' : '卖出'}
                    </span>
                  </div>
                  <div className="table-cell">¥{order.price.toFixed(2)}</div>
                  <div className="table-cell">{order.volume} kWh</div>
                  <div className="table-cell">{order.filledVolume} kWh</div>
                  <div className="table-cell">
                    <span className={`status-badge ${getStatusBadge(order.status).class}`}>
                      {getStatusBadge(order.status).text}
                    </span>
                  </div>
                  <div className="table-cell">{order.createTime}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'orderbook' && (
        <div className="card orderbook-card">
          <div className="card-header">
            <h3>订单簿</h3>
          </div>
          <div className="orderbook-container">
            <div className="orderbook-side buy-side">
              <div className="side-header">
                <span>买入</span>
              </div>
              <div className="orderbook-rows">
                {orderBook.filter(e => e.buyVolume > 0).reverse().map((entry, idx) => (
                  <div key={idx} className="orderbook-row buy">
                    <div className="price">¥{entry.price.toFixed(2)}</div>
                    <div className="volume">{entry.buyVolume.toLocaleString()}</div>
                    <div
                      className="volume-bar"
                      style={{ width: `${(entry.buyVolume / 15000) * 100}%` }}
                    />
                  </div>
                ))}
              </div>
            </div>
            <div className="orderbook-divider">
              <div className="current-price">
                <span className="label">最新价</span>
                <span className="price">¥0.44</span>
              </div>
            </div>
            <div className="orderbook-side sell-side">
              <div className="side-header">
                <span>卖出</span>
              </div>
              <div className="orderbook-rows">
                {orderBook.filter(e => e.sellVolume > 0).map((entry, idx) => (
                  <div key={idx} className="orderbook-row sell">
                    <div className="price">¥{entry.price.toFixed(2)}</div>
                    <div className="volume">{entry.sellVolume.toLocaleString()}</div>
                    <div
                      className="volume-bar"
                      style={{ width: `${(entry.sellVolume / 15000) * 100}%` }}
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'history' && (
        <div className="card history-card">
          <div className="card-header">
            <h3>交易历史</h3>
          </div>
          <div className="trades-table">
            <div className="table-header">
              <div className="table-cell">交易号</div>
              <div className="table-cell">价格</div>
              <div className="table-cell">数量</div>
              <div className="table-cell">时间</div>
              <div className="table-cell">买方</div>
              <div className="table-cell">卖方</div>
            </div>
            {trades.map(trade => (
              <div key={trade.id} className="table-row">
                <div className="table-cell">{trade.id}</div>
                <div className="table-cell">¥{trade.price.toFixed(2)}</div>
                <div className="table-cell">{trade.volume} kWh</div>
                <div className="table-cell">{trade.time}</div>
                <div className="table-cell">{trade.buyer}</div>
                <div className="table-cell">{trade.seller}</div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default TradingCenter;
