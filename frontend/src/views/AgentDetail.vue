<!--
  AgentDetail.vue
  
  服务器详情页面组件，用于展示单个代理服务器的详细信息和监控指标。
  
  主要功能：
  1. 展示服务器基本信息（主机名、IP地址、平台等）
  2. 显示当前CPU、内存、磁盘使用率的仪表盘
  3. 通过切换标签页展示不同类型的历史数据图表
  4. 支持选择不同时间范围（1小时、1天、1周）查看历史趋势
  5. 自动定时刷新数据保持实时性
  
  作者：Linux Monitor Team
  版本：1.0.0
-->

<template>
  <div class="agent-detail">
    <!-- 页面头部导航 -->
    <el-page-header @back="goBack" title="返回仪表盘" />
    
    <!-- 服务器基本信息卡片 -->
    <el-card v-loading="loading" class="agent-info-card">
      <template #header>
        <div class="card-header">
          <span>{{ agent.name || agent.hostname }}</span>
          <el-tag :type="agent.is_online ? 'success' : 'danger'">
            {{ agent.is_online ? '在线' : '离线' }}
          </el-tag>
        </div>
      </template>
      
      <!-- 服务器详细信息列表 -->
      <el-descriptions :column="2" border>
        <el-descriptions-item label="主机名">{{ agent.hostname }}</el-descriptions-item>
        <el-descriptions-item label="平台">{{ agent.platform }}</el-descriptions-item>
        <el-descriptions-item label="IP地址">{{ agent.ip_address }}</el-descriptions-item>
        <el-descriptions-item label="最后在线">
          {{ timeSince(agent.last_seen) }}
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">
          {{ formatDate(agent.created_at) }}
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
    
    <!-- 主要指标概览（CPU、内存、磁盘仪表盘） -->
    <el-row :gutter="20" class="metrics-row">
      <!-- CPU使用率 -->
      <el-col :span="8">
        <el-card class="metric-card">
          <template #header>
            <div class="card-header">
              <span>CPU使用率</span>
            </div>
          </template>
          <div v-if="agent.is_online" class="metric-content">
            <el-progress
              type="dashboard"
              :percentage="Number(latestMetrics.cpu_usage)"
              :status="getStatusByCpuUsage(Number(latestMetrics.cpu_usage))"
            />
            <div class="metric-info">
              <div class="metric-value">{{ latestMetrics.cpu_usage }}%</div>
              <div class="metric-label">当前使用率</div>
            </div>
          </div>
          <div v-else class="metric-content offline">
            <el-icon size="48"><WarningFilled /></el-icon>
            <div class="metric-info">
              <div class="metric-value">离线</div>
              <div class="metric-label">无法获取数据</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <!-- 内存使用率 -->
      <el-col :span="8">
        <el-card class="metric-card">
          <template #header>
            <div class="card-header">
              <span>内存使用率</span>
            </div>
          </template>
          <div v-if="agent.is_online" class="metric-content">
            <el-progress
              type="dashboard"
              :percentage="Number(latestMetrics.memory_percent)"
              :status="getStatusByUsage(Number(latestMetrics.memory_percent))"
            />
            <div class="metric-info">
              <div class="metric-value">{{ latestMetrics.memory_percent }}%</div>
              <div class="metric-label">当前使用率</div>
            </div>
          </div>
          <div v-else class="metric-content offline">
            <el-icon size="48"><WarningFilled /></el-icon>
            <div class="metric-info">
              <div class="metric-value">离线</div>
              <div class="metric-label">无法获取数据</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <!-- 磁盘使用率 -->
      <el-col :span="8">
        <el-card class="metric-card">
          <template #header>
            <div class="card-header">
              <span>磁盘使用率</span>
            </div>
          </template>
          <div v-if="agent.is_online" class="metric-content">
            <el-progress
              type="dashboard"
              :percentage="Number(latestMetrics.disk_percent)"
              :status="getStatusByUsage(Number(latestMetrics.disk_percent))"
            />
            <div class="metric-info">
              <div class="metric-value">{{ latestMetrics.disk_percent }}%</div>
              <div class="metric-label">当前使用率</div>
            </div>
          </div>
          <div v-else class="metric-content offline">
            <el-icon size="48"><WarningFilled /></el-icon>
            <div class="metric-info">
              <div class="metric-value">离线</div>
              <div class="metric-label">无法获取数据</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <!-- 历史数据图表部分 -->
    <el-card class="chart-card">
      <template #header>
        <div class="card-header">
          <span>历史数据</span>
          <!-- 时间范围选择 -->
          <el-radio-group v-model="timeRange" @change="fetchMetrics" :disabled="!agent.is_online">
            <el-radio-button :label="3600">1小时</el-radio-button>
            <el-radio-button :label="86400">1天</el-radio-button>
            <el-radio-button :label="604800">1周</el-radio-button>
          </el-radio-group>
        </div>
      </template>
      
      <!-- 代理离线提示 -->
      <div v-if="!agent.is_online" class="offline-message">
        <el-alert
          title="代理当前离线"
          type="warning"
          description="该代理当前处于离线状态，无法获取实时监控数据。当代理重新上线后，图表将自动更新。"
          show-icon
          :closable="false"
        />
      </div>
      
      <!-- 各类指标的图表标签页 -->
      <el-tabs v-model="activeTab" :disabled="!agent.is_online" @tab-click="handleTabClick">
        <!-- CPU使用率图表 -->
        <el-tab-pane label="CPU" name="cpu">
          <div id="cpu-chart" ref="cpuChart" class="chart"></div>
        </el-tab-pane>
        
        <!-- 内存使用率图表 -->
        <el-tab-pane label="内存" name="memory">
          <div id="memory-chart" ref="memoryChart" class="chart"></div>
        </el-tab-pane>
        
        <!-- 磁盘使用率图表 -->
        <el-tab-pane label="磁盘" name="disk">
          <div id="disk-chart" ref="diskChart" class="chart"></div>
        </el-tab-pane>
        
        <!-- 负载平均值图表 -->
        <el-tab-pane label="负载" name="load">
          <div id="load-chart" ref="loadChart" class="chart"></div>
        </el-tab-pane>
        
        <!-- 进程数图表 -->
        <el-tab-pane label="进程数" name="process">
          <div id="process-chart" ref="processChart" class="chart"></div>
        </el-tab-pane>
        
        <!-- 网络流量图表 -->
        <el-tab-pane label="网络流量" name="network">
          <div id="network-chart" ref="networkChart" class="chart"></div>
        </el-tab-pane>
        
        <!-- 网络连接数图表 -->
        <el-tab-pane label="网络连接" name="connections">
          <div id="connections-chart" ref="connectionsChart" class="chart"></div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
/**
 * AgentDetail 组件
 * 
 * 用于展示单个代理服务器的详细监控信息和历史数据图表
 */

import { ref, computed, onMounted, onBeforeUnmount, watch, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useStore } from 'vuex'
import * as echarts from 'echarts'  // 图表库
import agentApi from '@/api/agent'  // 代理API接口
import { ElMessage } from 'element-plus'
import { WarningFilled } from '@element-plus/icons-vue'

// 路由和状态管理
const route = useRoute()
const router = useRouter()
const store = useStore()

// 响应式状态变量
const loading = ref(false)        // 加载状态
const agent = ref({})             // 代理信息
const metrics = ref([])           // 指标数据
const timeRange = ref(604800)     // 时间范围(秒)，默认显示7天数据
const activeTab = ref('cpu')      // 当前激活的标签页
const charts = ref({})            // 图表实例集合
const refreshInterval = ref(null) // 自动刷新定时器

// 图表容器引用
const cpuChart = ref(null)         // CPU图表容器引用
const memoryChart = ref(null)      // 内存图表容器引用
const diskChart = ref(null)        // 磁盘图表容器引用
const loadChart = ref(null)        // 负载图表容器引用
const processChart = ref(null)     // 进程图表容器引用
const networkChart = ref(null)     // 网络图表容器引用
const connectionsChart = ref(null) // 连接数图表容器引用

/**
 * 计算最新指标数据
 * 从指标数据数组中提取最新的指标，用于显示在仪表盘上
 */
const latestMetrics = computed(() => {
  // 如果指标数据为空或者代理离线，返回默认值
  if (!metrics.value || !metrics.value.length || !agent.value || !agent.value.is_online) {
    console.log('没有可用的指标数据或代理离线，返回默认值');
    return {
    cpu_usage: 0,
    memory_percent: 0,
    disk_percent: 0,
      load_average: { load1: 0, load5: 0, load15: 0 },
      process_count: 0,
      network_info: { bytes_sent: 0, bytes_recv: 0, tcp_connections: 0, udp_connections: 0 }
    }
  }
  
  // 获取最新的指标数据（数组中的第一个元素）
  const latest = metrics.value[0];
  
  // 安全地提取百分比数据并转换为数字
  const safeExtractPercent = (obj, path) => {
    if (!obj) return 0;
    
    // 嵌套属性访问
    const pathParts = path.split('.');
    let current = obj;
    
    for (const part of pathParts) {
      if (current === null || current === undefined) {
        return 0;
      }
      current = current[part];
    }
    
    const value = parseFloat(current);
    return isNaN(value) ? 0 : Number(value.toFixed(1));
  };
  
  // 提取并格式化CPU使用率 - 兼容多种字段名
  let cpuUsage = safeExtractPercent(latest, 'cpu_usage');
  if (cpuUsage === 0) {
    cpuUsage = safeExtractPercent(latest, 'cpu_percent');
  }
  
  // 提取并格式化内存使用率 - 兼容多种字段结构
  let memoryPercent = safeExtractPercent(latest, 'memory_info.percent');
  if (memoryPercent === 0) {
    memoryPercent = safeExtractPercent(latest, 'memory_percent');
  }
  
  // 提取并格式化磁盘使用率 - 兼容多种字段结构
  let diskPercent = safeExtractPercent(latest, 'disk_info.percent');
  if (diskPercent === 0) {
    diskPercent = safeExtractPercent(latest, 'disk_percent');
  }
  
  // 为控制台输出基本信息
  console.log(`最新指标数据: CPU=${cpuUsage}%, 内存=${memoryPercent}%, 磁盘=${diskPercent}%`);
  
  // 兼容不同的数据结构
  const loadAverage = latest.load_average || 
                      { load1: latest.load1 || 0, 
                        load5: latest.load5 || 0, 
                        load15: latest.load15 || 0 };
                      
  const networkInfo = latest.network_info || 
                     { bytes_sent: latest.bytes_sent || 0, 
                       bytes_recv: latest.bytes_recv || 0, 
                       tcp_connections: latest.tcp_connections || 0, 
                       udp_connections: latest.udp_connections || 0 };
  
  return {
    cpu_usage: cpuUsage,
    memory_percent: memoryPercent,
    disk_percent: diskPercent,
    load_average: loadAverage,
    process_count: latest.process_count || 0,
    network_info: networkInfo
  }
})

// 获取状态
const getStatusByCpuUsage = (usage) => {
  if (usage >= 90) return 'exception'
  if (usage >= 70) return 'warning'
  return 'success'
}

const getStatusByUsage = (usage) => {
  if (usage >= 90) return 'exception'
  if (usage >= 70) return 'warning'
  return 'success'
}

// 格式化日期时间
const formatDate = (date) => {
  return date.getFullYear() + '-' +
         (date.getMonth() + 1).toString().padStart(2, '0') + '-' +
         date.getDate().toString().padStart(2, '0') + ' ' +
         date.getHours().toString().padStart(2, '0') + ':' +
         date.getMinutes().toString().padStart(2, '0') + ':' +
         date.getSeconds().toString().padStart(2, '0');
};

// 时间差计算
const timeSince = (date) => {
  if (!date) return '未知';
  
  if (typeof date === 'string') {
    date = new Date(date);
  }
  
  const seconds = Math.floor((new Date() - date) / 1000);
  
  let interval = seconds / 31536000;
  if (interval > 1) return Math.floor(interval) + ' 年前';
  
  interval = seconds / 2592000;
  if (interval > 1) return Math.floor(interval) + ' 个月前';
  
  interval = seconds / 86400;
  if (interval > 1) return Math.floor(interval) + ' 天前';
  
  interval = seconds / 3600;
  if (interval > 1) return Math.floor(interval) + ' 小时前';
  
  interval = seconds / 60;
  if (interval > 1) return Math.floor(interval) + ' 分钟前';
  
  return Math.floor(seconds) + ' 秒前';
};

// 返回上一页
const goBack = () => {
  router.push('/')
}

// 获取代理详情
const fetchAgent = async () => {
  try {
    // 获取ID参数并移除可能存在的花括号
    let agentId = route.params.id;
    if (agentId.startsWith('{') && agentId.endsWith('}')) {
      agentId = agentId.substring(1, agentId.length - 1);
      // 更新URL以移除花括号，但不触发新的导航
      const newPath = `/agent/${agentId}`;
      console.log(`将URL从 ${route.fullPath} 重定向到 ${newPath}`);
      router.replace(newPath);
    }
    
    console.log(`获取代理详情，ID: ${agentId}`);
    
    // 首先检查 store 中是否已有该代理数据
    const storeAgents = store.getters.agents;
    const storeAgent = storeAgents.find(a => a.id === agentId);
    
    if (storeAgent) {
      console.log(`从 store 中获取到代理信息: ${storeAgent.name || storeAgent.hostname}`);
      agent.value = storeAgent;
    } else {
      // 如果 store 中没有，则单独获取代理信息
      console.log(`从 API 中获取代理信息`);
      const response = await agentApi.getAgent(agentId);
      
      // 处理响应数据
      if (response && response.data) {
        // 如果数据在 data 字段中，取出并赋值
        if (response.data.data) {
          agent.value = response.data.data;
        } else {
          agent.value = response.data;
        }
      }
    }
    
    if (!agent.value || !agent.value.id) {
      throw new Error('无法获取有效的代理信息');
    }
    
    // 确保必要的字段存在
    if (!agent.value.is_online && agent.value.isOnline !== undefined) {
      agent.value.is_online = agent.value.isOnline;
    }
    
    console.log('获取到代理信息:', agent.value);
  } catch (error) {
    console.error('获取代理详情失败:', error);
    agent.value = {};
    ElMessage.error('获取代理详情失败: ' + error.message);
  }
}

// 获取指标数据
const fetchMetrics = async () => {
  try {
    // 如果代理未获取或离线，则不获取指标
    if (!agent.value || !agent.value.id) {
      console.warn('代理未获取或ID无效，无法获取指标');
      return;
    }
    
    if (!agent.value.is_online) {
      console.warn('代理离线，不获取指标');
      metrics.value = [];
      return;
    }
    
    let agentId = agent.value.id;
    const now = Math.floor(Date.now() / 1000);
    const from = now - timeRange.value;
    
    console.log(`获取指标数据，ID: ${agentId}，时间范围: ${timeRange.value}秒，从 ${from} 到 ${now}`);
    
    // 当请求七天数据时增加数据点数量
    const limit = timeRange.value === 604800 ? 300 : (timeRange.value === 86400 ? 150 : 100);
    
    const response = await agentApi.getAgentMetrics(agentId, {
      from,
      to: now,
      limit: limit
    });
    
    // 检查返回数据结构，处理不同格式的返回结果
    if (response) {
      let metricsData = [];
      
      // 处理直接返回数组的情况
      if (Array.isArray(response)) {
        metricsData = response;
      }
      // 处理响应是直接数组的情况
      else if (Array.isArray(response.data)) {
        metricsData = response.data;
      }
      // 处理响应对象中包含数组的情况
      else if (response.data && response.data.data && Array.isArray(response.data.data)) {
        metricsData = response.data.data;
      }
      
      console.log(`成功获取到 ${metricsData.length} 条数据点`);
      metrics.value = metricsData;
      
      // 获取数据后更新当前活动标签的图表
      if (metricsData.length > 0 && agent.value.is_online) {
        const tab = { name: activeTab.value };
        handleTabClick(tab);
      }
    } else {
      console.warn('API返回的数据格式不符合预期:', response);
      metrics.value = [];
    }
  } catch (error) {
    console.error('获取指标数据失败:', error);
    metrics.value = [];
    ElMessage.error('获取指标数据失败: ' + error.message);
  }
}

// 清空图表
const clearCharts = () => {
  console.log(`开始清理所有图表实例，当前实例数: ${Object.keys(charts.value).length}`);
  
  Object.entries(charts.value).forEach(([id, chart]) => {
    if (chart) {
      console.log(`清理图表实例: ${id}`);
      
      // 移除该图表注册的resize事件监听器
      if (chart._resizeListener) {
        window.removeEventListener('resize', chart._resizeListener);
        console.log(`已移除图表 ${id} 的resize事件监听器`);
      }
      
      // 销毁图表实例
      chart.dispose();
    }
  });
  
  // 重置图表实例集合
  charts.value = {};
  console.log('所有图表实例已清理完毕');
}

// 计算网络速率 (KB/s)
const calculateNetworkRate = (dataPoints) => {
  if (!dataPoints || dataPoints.length < 2) return [];
  
  const rateData = [];
  for (let i = 1; i < dataPoints.length; i++) {
    const currentPoint = dataPoints[i];
    const prevPoint = dataPoints[i-1];
    
    const timeDiff = (currentPoint[0] - prevPoint[0]) / 1000; // 时间差（秒）
    if (timeDiff <= 0) continue;
    
    const byteDiff = currentPoint[1] - prevPoint[1]; // 字节差
    // 如果字节差为负数，可能是计数器重置，此时跳过该点
    if (byteDiff < 0) continue;
    
    // 计算每秒字节数，并转换为 KB/s
    const rateKBs = (byteDiff / timeDiff) / 1024;
    rateData.push([currentPoint[0], rateKBs]);
  }
  
  return rateData;
};

// 格式化网络流量显示
const formatNetworkTraffic = (bytes) => {
  if (bytes < 1024) {
    return bytes + ' B';
  } else if (bytes < 1024 * 1024) {
    return (bytes / 1024).toFixed(2) + ' KB';
  } else if (bytes < 1024 * 1024 * 1024) {
    return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
  } else {
    return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
  }
};

// 为指定标签初始化图表
const initChartForTab = (tabName, chartDom) => {
  console.log(`开始为标签 ${tabName} 创建图表...`);
  
  // 确保图表容器尺寸已正确计算
  if (chartDom.clientWidth === 0 || chartDom.clientHeight === 0) {
    console.warn(`图表容器尺寸为零: width=${chartDom.clientWidth}, height=${chartDom.clientHeight}，尝试强制设置尺寸`);
    chartDom.style.width = '100%';
    chartDom.style.height = '400px';
    // 强制触发重排以确保尺寸计算正确
    void chartDom.offsetHeight;
  }
  
  console.log(`图表容器尺寸: width=${chartDom.clientWidth}, height=${chartDom.clientHeight}`);
  
  // 为当前标签创建新的图表实例
  const chart = echarts.init(chartDom, null, {
    renderer: 'canvas',
    useDirtyRect: false
  });
  
  charts.value[chartDom.id] = chart;
  
  // 设置loading状态
  chart.showLoading({
    text: '加载数据中...',
    maskColor: 'rgba(255, 255, 255, 0.8)',
    fontSize: 16
  });
  
  // 仅为当前标签页准备数据并渲染图表
  if (metrics.value && metrics.value.length > 0) {
    try {
      // 使用自定义函数创建适合当前图表类型的配置
      createChartOption(tabName, chart, metrics.value);
      chart.hideLoading();
      console.log(`${tabName}图表创建成功`);
      
      // 强制执行一次重置大小操作，确保图表正确适应容器
      setTimeout(() => {
        console.log(`强制调整${tabName}图表大小`);
        chart.resize({width: 'auto', height: 'auto'});
      }, 100);
    } catch (e) {
      console.error(`为 ${tabName} 创建图表时出错:`, e);
      chart.hideLoading();
    }
  } else {
    console.warn('没有可用的指标数据，无法绘制图表');
    chart.hideLoading();
    chart.setOption({
      title: {
        text: '没有数据可显示',
        left: 'center',
        top: 'center',
        textStyle: {
          fontSize: 20,
          color: '#999'
        }
      }
    });
  }
  
  // 添加窗口大小变化监听器
  const resizeListener = () => {
    console.log(`窗口大小变化，调整${tabName}图表大小`);
    chart.resize();
  };
  
  window.addEventListener('resize', resizeListener);
  
  // 保存事件监听器引用以便后续清理
  chart._resizeListener = resizeListener;
  
  console.log(`标签 ${tabName} 的图表已创建完成`);
  
  return chart;
}

// 处理标签页点击事件
const handleTabClick = (tab) => {
  console.log('标签点击：', tab);
  
  // 获取标签名称，兼容不同的对象结构
  const tabName = tab.props?.name || tab.paneName || tab.name;
  
  if (tabName) {
    // 先清除所有图表，释放资源
    clearCharts();
    
    // 更新活动标签
    activeTab.value = tabName;
    console.log('活动标签已更新为：', activeTab.value);
    
    // 使用 nextTick 确保DOM已完全更新
    nextTick(() => {
      // 根据当前活动标签获取对应的图表引用
      const chartRefs = {
        'cpu': cpuChart,
        'memory': memoryChart,
        'disk': diskChart,
        'load': loadChart,
        'process': processChart,
        'network': networkChart,
        'connections': connectionsChart
      };
      
      const chartDom = chartRefs[activeTab.value].value;
      
      if (chartDom) {
        console.log(`找到图表容器：${chartDom.id}，宽度：${chartDom.clientWidth}`);
        // 初始化新的图表
        initChartForTab(activeTab.value, chartDom);
      } else {
        console.error(`找不到图表容器：${activeTab.value}`);
      }
    });
  } else {
    console.error('无效的标签点击事件：', tab);
  }
}

// 创建通用的X轴时间轴配置
const createTimeXAxis = () => {
  return {
    type: 'time',
    splitLine: {
      show: false
    },
    axisPointer: {
      show: true
    },
    axisLine: {
      show: true
    },
    axisTick: {
      show: true
    },
    boundaryGap: false,
    // 设置合理的最小宽度，防止轴线太短
    min: function(value) {
      // 如果数据点少于3个，扩展一下显示范围
      if (value.min === value.max) {
        return value.min - 3600000; // 减少1小时
      }
      return value.min;
    },
    max: function(value) {
      // 如果数据点少于3个，扩展一下显示范围
      if (value.min === value.max) {
        return value.max + 3600000; // 增加1小时
      }
      return value.max;
    },
    axisLabel: {
      formatter: function(value) {
        const date = new Date(value);
        return date.getMonth() + 1 + '/' + date.getDate() + ' ' + 
               date.getHours() + ':' + date.getMinutes();
      },
      // 确保标签有足够空间显示
      interval: 'auto',
      rotate: 0
    }
  };
};

// 为指定图表类型创建配置选项
const createChartOption = (chartType, chart, dataPoints) => {
  if (!chart || !dataPoints || dataPoints.length < 2) {
    console.warn(`为${chartType}创建图表配置失败：图表实例不存在或数据不足`);
    return;
  }
  
  // 获取数据点数量，并对数据进行排序和处理
  const sortedData = [...dataPoints].sort((a, b) => a.timestamp - b.timestamp);
  console.log(`创建${chartType}图表配置，共 ${sortedData.length} 个数据点`);
  
  // 设置时间范围标题
  let timeRangeTitle = '';
  if (timeRange.value === 3600) {
    timeRangeTitle = '过去1小时';
  } else if (timeRange.value === 86400) {
    timeRangeTitle = '过去1天';
  } else if (timeRange.value === 604800) {
    timeRangeTitle = '过去7天';
  }
  
  // 获取通用的X轴配置
  const xAxisConfig = createTimeXAxis();
  
  // 根据图表类型准备数据和配置
  let option;
  
  if (chartType === 'cpu') {
    // CPU 使用率数据
    const cpuData = sortedData.map(m => {
      const cpuValue = parseFloat(m.cpu_usage || m.cpu_percent || 0);
      return [m.timestamp * 1000, cpuValue];
    });
    
    option = {
      title: {
        text: `CPU使用率 (${timeRangeTitle})`,
        left: 'center'
      },
    tooltip: {
        trigger: 'axis',
        formatter: function(params) {
          const date = new Date(params[0].value[0]);
          return formatDate(date) + '<br />' +
                 'CPU使用率: ' + params[0].value[1].toFixed(2) + '%';
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: xAxisConfig,
    yAxis: {
      type: 'value',
      min: 0,
        max: 100,
        axisLabel: {
          formatter: '{value}%'
        }
    },
    series: [{
        name: 'CPU使用率',
        data: cpuData,
      type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: {}
      }]
    };
  }
  else if (chartType === 'memory') {
    // 内存使用率数据
    const memoryData = sortedData.map(m => {
      let memValue = 0;
      if (m.memory_info && m.memory_info.percent) {
        memValue = parseFloat(m.memory_info.percent);
      } else if (m.memory_percent) {
        memValue = parseFloat(m.memory_percent);
      }
      return [m.timestamp * 1000, memValue];
    });
    
    option = {
      title: {
        text: `内存使用率 (${timeRangeTitle})`,
        left: 'center'
      },
    tooltip: {
        trigger: 'axis',
        formatter: function(params) {
          const date = new Date(params[0].value[0]);
          return formatDate(date) + '<br />' +
                 '内存使用率: ' + params[0].value[1].toFixed(2) + '%';
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: xAxisConfig,
    yAxis: {
      type: 'value',
      min: 0,
        max: 100,
        axisLabel: {
          formatter: '{value}%'
        }
    },
    series: [{
        name: '内存使用率',
        data: memoryData,
      type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: {}
      }]
    };
  }
  else if (chartType === 'disk') {
    // 磁盘使用率数据
    const diskData = sortedData.map(m => {
      let diskValue = 0;
      if (m.disk_info && m.disk_info.percent) {
        diskValue = parseFloat(m.disk_info.percent);
      } else if (m.disk_percent) {
        diskValue = parseFloat(m.disk_percent);
      }
      return [m.timestamp * 1000, diskValue];
    });
    
    option = {
      title: {
        text: `磁盘使用率 (${timeRangeTitle})`,
        left: 'center'
      },
    tooltip: {
        trigger: 'axis',
        formatter: function(params) {
          const date = new Date(params[0].value[0]);
          return formatDate(date) + '<br />' +
                 '磁盘使用率: ' + params[0].value[1].toFixed(2) + '%';
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: xAxisConfig,
    yAxis: {
      type: 'value',
      min: 0,
        max: 100,
        axisLabel: {
          formatter: '{value}%'
        }
    },
    series: [{
        name: '磁盘使用率',
        data: diskData,
      type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: {}
      }]
    };
  }
  else if (chartType === 'load') {
    // 负载数据
    const loadData1 = sortedData.map(m => {
      let load1 = 0;
      if (m.load_average && m.load_average.load1) {
        load1 = parseFloat(m.load_average.load1);
      } else if (m.load1) {
        load1 = parseFloat(m.load1);
      }
      return [m.timestamp * 1000, load1];
    });
    
    const loadData5 = sortedData.map(m => {
      let load5 = 0;
      if (m.load_average && m.load_average.load5) {
        load5 = parseFloat(m.load_average.load5);
      } else if (m.load5) {
        load5 = parseFloat(m.load5);
      }
      return [m.timestamp * 1000, load5];
    });
    
    const loadData15 = sortedData.map(m => {
      let load15 = 0;
      if (m.load_average && m.load_average.load15) {
        load15 = parseFloat(m.load_average.load15);
      } else if (m.load15) {
        load15 = parseFloat(m.load15);
      }
      return [m.timestamp * 1000, load15];
    });
    
    option = {
      title: {
        text: `系统负载 (${timeRangeTitle})`,
        left: 'center'
      },
    tooltip: {
        trigger: 'axis',
        formatter: function(params) {
          const date = new Date(params[0].value[0]);
          let result = formatDate(date) + '<br />';
          params.forEach(param => {
            result += param.seriesName + ': ' + param.value[1].toFixed(2) + '<br />';
          });
          return result;
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '10%',
        containLabel: true
      },
      legend: {
        data: ['1分钟', '5分钟', '15分钟'],
        bottom: 0
      },
      xAxis: xAxisConfig,
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '1分钟',
          data: loadData1,
        type: 'line',
        smooth: true
      },
      {
        name: '5分钟',
          data: loadData5,
        type: 'line',
        smooth: true
      },
      {
        name: '15分钟',
          data: loadData15,
        type: 'line',
        smooth: true
      }
    ]
    };
  }
  else if (chartType === 'process') {
    // 进程数据
    const processData = sortedData.map(m => 
      [m.timestamp * 1000, parseInt(m.process_count || 0)]
    );
    
    option = {
      title: {
        text: `进程数 (${timeRangeTitle})`,
        left: 'center'
      },
      tooltip: {
        trigger: 'axis',
        formatter: function(params) {
          const date = new Date(params[0].value[0]);
          return formatDate(date) + '<br />' +
                 '进程数: ' + params[0].value[1];
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: xAxisConfig,
      yAxis: {
        type: 'value',
        minInterval: 1
      },
      series: [{
        name: '进程数',
        data: processData,
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: {}
      }]
    };
  }
  else if (chartType === 'network') {
    // 网络数据
    const networkSentData = sortedData.map(m => {
      let bytesSent = 0;
      if (m.network_info && m.network_info.bytes_sent) {
        bytesSent = parseInt(m.network_info.bytes_sent);
      } else if (m.bytes_sent) {
        bytesSent = parseInt(m.bytes_sent);
      }
      return [m.timestamp * 1000, bytesSent];
    });
    
    const networkRecvData = sortedData.map(m => {
      let bytesRecv = 0;
      if (m.network_info && m.network_info.bytes_recv) {
        bytesRecv = parseInt(m.network_info.bytes_recv);
      } else if (m.bytes_recv) {
        bytesRecv = parseInt(m.bytes_recv);
      }
      return [m.timestamp * 1000, bytesRecv];
    });
    
    // 计算网络速率
    const sentRateData = calculateNetworkRate(networkSentData);
    const recvRateData = calculateNetworkRate(networkRecvData);
    
    option = {
      title: {
        text: `网络流量 (${timeRangeTitle})`,
        left: 'center'
      },
      tooltip: {
        trigger: 'axis',
        formatter: function(params) {
          const date = new Date(params[0].value[0]);
          let result = formatDate(date) + '<br />';
          params.forEach(param => {
            const value = param.value[1];
            const formattedValue = param.seriesIndex < 2 
              ? formatNetworkTraffic(value) 
              : (value / 1024).toFixed(2) + ' KB/s';
            result += param.seriesName + ': ' + formattedValue + '<br />';
          });
          return result;
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '10%',
        containLabel: true
      },
      legend: {
        data: ['发送总量', '接收总量', '发送速率', '接收速率'],
        bottom: 0
      },
      xAxis: xAxisConfig,
      yAxis: [
        {
          type: 'value',
          name: '流量总量',
          axisLabel: {
            formatter: function(value) {
              return formatNetworkTraffic(value);
            }
          }
        },
        {
          type: 'value',
          name: '速率 (KB/s)',
          axisLabel: {
            formatter: '{value} KB/s'
          }
        }
      ],
      series: [
        {
          name: '发送总量',
          data: networkSentData,
          type: 'line',
          smooth: true,
          yAxisIndex: 0
        },
        {
          name: '接收总量',
          data: networkRecvData,
          type: 'line',
          smooth: true,
          yAxisIndex: 0
        },
        {
          name: '发送速率',
          data: sentRateData,
          type: 'line',
          smooth: true,
          yAxisIndex: 1
        },
        {
          name: '接收速率',
          data: recvRateData,
          type: 'line',
          smooth: true,
          yAxisIndex: 1
        }
      ]
    };
  }
  else if (chartType === 'connections') {
    // 网络连接数据
    const tcpConnectionsData = sortedData.map(m => {
      const timestamp = m.timestamp * 1000;
      let connections = 0;
      
      if (m.network_info && typeof m.network_info.tcp_connections !== 'undefined') {
        connections = parseInt(m.network_info.tcp_connections);
      } else if (typeof m.tcp_connections !== 'undefined') {
        connections = parseInt(m.tcp_connections);
      }
      
      if (isNaN(connections)) connections = 0;
      return [timestamp, connections];
    });
    
    const udpConnectionsData = sortedData.map(m => {
      const timestamp = m.timestamp * 1000;
      let connections = 0;
      
      if (m.network_info && typeof m.network_info.udp_connections !== 'undefined') {
        connections = parseInt(m.network_info.udp_connections);
      } else if (typeof m.udp_connections !== 'undefined') {
        connections = parseInt(m.udp_connections);
      }
      
      if (isNaN(connections)) connections = 0;
      return [timestamp, connections];
    });
    
    option = {
      title: {
        text: `网络连接数 (${timeRangeTitle})`,
        left: 'center'
      },
      tooltip: {
        trigger: 'axis',
        formatter: function(params) {
          const date = new Date(params[0].value[0]);
          let result = formatDate(date) + '<br />';
          params.forEach(param => {
            result += param.seriesName + ': ' + param.value[1] + '<br />';
          });
          return result;
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '10%',
        containLabel: true
      },
      legend: {
        data: ['TCP连接数', 'UDP连接数'],
        bottom: 0
      },
      xAxis: xAxisConfig,
      yAxis: {
        type: 'value',
        minInterval: 1
      },
      series: [
        {
          name: 'TCP连接数',
          data: tcpConnectionsData,
          type: 'line',
          smooth: true,
          showSymbol: false,
          areaStyle: {}
        },
        {
          name: 'UDP连接数',
          data: udpConnectionsData,
          type: 'line',
          smooth: true,
          showSymbol: false,
          areaStyle: {}
        }
      ]
    };
  }
  
  // 设置图表选项
  if (option) {
    chart.setOption(option, true);
  } else {
    console.warn(`没有为${chartType}图表找到配置`);
  }
}

// 生命周期钩子
onMounted(async () => {
  console.log('组件挂载，初始化数据和图表，路由参数:', route.params);
  loading.value = true;
  
  try {
    // 添加窗口大小变化监听
    window.addEventListener('resize', handleResize);
    
    // 确保 store 中有最新的代理列表
    await store.dispatch('fetchAgents');
    
    // 获取代理信息
    await fetchAgent();
    
    if (!agent.value || !agent.value.id) {
      console.error('无法获取代理信息或代理ID无效，当前路由参数:', route.params);
      loading.value = false;
      return;
    }
  
    console.log(`成功获取代理信息: ID=${agent.value.id}, 名称=${agent.value.name || agent.value.hostname}`);
    
    // 获取代理指标数据
    await fetchMetrics();
    
    // 使用延迟确保DOM已完全加载，然后初始化当前活动标签的图表
    nextTick(() => {
      console.log('页面初始化完成，开始创建初始图表');
      if (agent.value.is_online && metrics.value.length > 0) {
        // 默认初始化第一个图表（CPU图表）
        const cpuChartDom = cpuChart.value;
        if (cpuChartDom) {
          console.log('初始化CPU图表');
          initChartForTab('cpu', cpuChartDom);
        } else {
          console.error('找不到CPU图表容器');
        }
      } else {
        console.warn('代理离线或无指标数据，不创建初始图表');
      }
    });
  
  // 设置定时刷新
    refreshInterval.value = setInterval(async () => {
      console.log('定时刷新数据');
      // 刷新代理列表，确保状态最新
      await store.dispatch('fetchAgents');
      
      // 重新获取代理信息
      await fetchAgent();
      
      // 如果代理在线，则获取指标
      if (agent.value && agent.value.is_online) {
        await fetchMetrics();
        
        // 刷新当前活动标签的图表
        nextTick(() => {
          const chartRefs = {
            'cpu': cpuChart,
            'memory': memoryChart,
            'disk': diskChart,
            'load': loadChart,
            'process': processChart,
            'network': networkChart,
            'connections': connectionsChart
          };
          
          const chartDom = chartRefs[activeTab.value].value;
          if (chartDom) {
            // 清除所有图表
            clearCharts();
            // 重新创建当前活动标签的图表
            initChartForTab(activeTab.value, chartDom);
          }
        });
      }
    }, 30000); // 30秒刷新一次
  } catch (error) {
    console.error('初始化错误:', error);
    ElMessage.error('初始化错误: ' + error.message);
  } finally {
    loading.value = false;
  }
});

onUnmounted(() => {
  console.log('组件卸载，清理资源')
  
  // 清除定时刷新
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
  
  // 移除窗口大小变化监听
  window.removeEventListener('resize', handleResize)
  
  // 清除所有图表实例
  clearCharts()
  
  console.log('所有资源已清理完毕')
})

// 处理窗口大小变化
const handleResize = () => {
  console.log('窗口大小变化，调整图表');
  
  // 仅处理当前活动标签的图表，避免不必要的计算
  handleTabChange(activeTab.value, false);
}

// 处理标签页变化
const handleTabChange = (tabName, forceRecreate = false) => {
  // 根据当前活动标签获取对应的图表引用
  const chartRefs = {
    'cpu': cpuChart,
    'memory': memoryChart,
    'disk': diskChart,
    'load': loadChart,
    'process': processChart,
    'network': networkChart,
    'connections': connectionsChart
  };
  
  const activeChartRef = chartRefs[tabName];
  
  if (activeChartRef && activeChartRef.value) {
    const dom = activeChartRef.value;
    const chart = charts.value[dom.id];
    
    if (chart && !forceRecreate) {
      console.log(`重新调整图表大小: ${tabName}`);
      chart.resize();
    } else if (forceRecreate) {
      console.log(`强制重新初始化图表: ${tabName}`);
      // 模拟点击标签页来重新创建图表
      const tab = { name: tabName };
      handleTabClick(tab);
    }
  }
}

// 监听活动标签页变化
watch(activeTab, (newTab, oldTab) => {
  console.log(`标签页监听变化: ${oldTab} -> ${newTab}`);
  
  // 检查是否是el-tabs组件手动触发的变化
  // 如果是通过tab-click事件改变的，handleTabClick已经处理
  // 如果是通过其他方式(如编程方式)改变的，需要手动处理图表
  
  // 为了防止重复处理，这里我们简单判断一下：
  // 如果oldTab存在（即不是首次加载），并且有图表实例，说明可能是编程方式切换的
  if (oldTab && newTab && newTab !== oldTab && Object.keys(charts.value).length === 0) {
    console.log(`通过编程方式切换标签: ${oldTab} -> ${newTab}，需要手动处理图表`);
    
    // 模拟点击标签页，触发图表创建
    const tab = { name: newTab };
    handleTabClick(tab);
  }
})

// 监听时间范围变化，重新加载数据
watch(timeRange, (newValue, oldValue) => {
  if (newValue !== oldValue) {
    console.log(`时间范围变化: ${oldValue} -> ${newValue}`);
    
    // 切换时间轴时，先清空所有图表实例，完全释放资源
    clearCharts();
    
    // 获取新的数据，然后重建当前活动标签的图表
    fetchMetrics().then(() => {
      console.log(`时间范围变化后获取到新数据，准备重建当前活动标签(${activeTab.value})的图表`);
      
      // 使用 nextTick 确保DOM已完全更新
      nextTick(() => {
        // 根据当前活动标签获取对应的图表引用
        const chartRefs = {
          'cpu': cpuChart,
          'memory': memoryChart,
          'disk': diskChart,
          'load': loadChart,
          'process': processChart,
          'network': networkChart,
          'connections': connectionsChart
        };
        
        const chartDom = chartRefs[activeTab.value].value;
        if (chartDom) {
          // 初始化新的图表
          initChartForTab(activeTab.value, chartDom);
        } else {
          console.error(`找不到图表容器：${activeTab.value}`);
        }
      });
    }).catch(error => {
      console.error('获取新时间范围数据失败:', error);
      ElMessage.error('获取数据失败: ' + error.message);
    });
  }
})
</script>

<style scoped>
.agent-detail {
  padding: 20px;
}

.agent-info-card {
  margin: 20px 0;
}

.metrics-row {
  margin: 20px 0;
}

.metric-card {
  height: 100%;
}

.metric-content {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.metric-content.offline {
  color: #E6A23C;
}

.metric-info {
  margin-left: 20px;
  text-align: center;
}

.metric-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.metric-content.offline .metric-value {
  color: #E6A23C;
}

.metric-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.chart-card {
  margin-top: 20px;
}

.chart {
  height: 400px;
  width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.offline-message {
  margin-bottom: 20px;
}
</style> 