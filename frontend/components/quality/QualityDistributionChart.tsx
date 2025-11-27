"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DynamicPieChart, DynamicPie, DynamicCell, DynamicBarChart, DynamicBar, ResponsiveContainer, Tooltip, Legend, XAxis, YAxis, CartesianGrid } from '@/lib/recharts-dynamic';
import { Badge } from "@/components/ui/badge";

const CONFIDENCE_COLORS = {
  '0.9-1.0': '#10b981', // green
  '0.7-0.9': '#3b82f6', // blue
  '0.5-0.7': '#f59e0b', // amber
  '0.3-0.5': '#f97316', // orange
  '0.0-0.3': '#ef4444', // red
};

interface QualityDistribution {
  range: string;
  count: number;
  percentage: number;
}

interface QualityDistributionChartProps {
  data: QualityDistribution[];
  totalRecords?: number;
  viewType?: 'pie' | 'bar';
}

export function QualityDistributionChart({ data, totalRecords, viewType = 'pie' }: QualityDistributionChartProps) {
  // Фильтруем данные, убирая записи с нулевыми значениями и NaN
  const validData = data.filter(item => 
    !isNaN(item.count) && 
    !isNaN(item.percentage) && 
    item.count > 0
  )
  
  const chartData = validData.map(item => ({
    name: item.range,
    value: item.count,
    percentage: item.percentage,
  }));
  
  // Если нет данных для отображения, показываем сообщение
  if (chartData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Распределение уверенности классификации</CardTitle>
          <CardDescription>
            Какой процент записей имеет различный уровень уверенности
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-[350px] text-muted-foreground">
            <p>Нет данных для отображения</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  // Кастомный тултип
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-white dark:bg-gray-800 p-3 rounded-lg shadow-lg border">
          <p className="font-medium">{data.name}</p>
          <div className="space-y-1 text-sm mt-2">
            <p>
              Записей: <span className="font-medium">{data.value.toLocaleString()}</span>
            </p>
            <p>
              Процент: <span className="font-medium">{data.percentage.toFixed(1)}%</span>
            </p>
          </div>
        </div>
      );
    }
    return null;
  };

  // Кастомная метка для pie chart
  const renderCustomLabel = ({
    cx,
    cy,
    midAngle,
    innerRadius,
    outerRadius,
    percentage,
  }: any) => {
    const RADIAN = Math.PI / 180;
    const radius = innerRadius + (outerRadius - innerRadius) * 0.5;
    const x = cx + radius * Math.cos(-midAngle * RADIAN);
    const y = cy + radius * Math.sin(-midAngle * RADIAN);

    if (percentage < 5) return null; // Hide labels for small slices

    return (
      <text
        x={x}
        y={y}
        fill="white"
        textAnchor={x > cx ? 'start' : 'end'}
        dominantBaseline="central"
        className="text-xs font-medium"
      >
        {`${percentage.toFixed(0)}%`}
      </text>
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Распределение уверенности классификации</CardTitle>
        <CardDescription>
          Какой процент записей имеет различный уровень уверенности
          {totalRecords && ` • Всего записей: ${totalRecords.toLocaleString()}`}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {viewType === 'pie' ? (
          <ResponsiveContainer width="100%" height={350}>
            <DynamicPieChart>
              <DynamicPie
                data={chartData}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={renderCustomLabel}
                outerRadius={120}
                fill="#8884d8"
                dataKey="value"
              >
                {chartData.map((entry, index) => (
                  <DynamicCell
                    key={`cell-${index}`}
                    fill={CONFIDENCE_COLORS[entry.name as keyof typeof CONFIDENCE_COLORS]}
                  />
                ))}
              </DynamicPie>
              <Tooltip content={<CustomTooltip />} />
              <Legend />
            </DynamicPieChart>
          </ResponsiveContainer>
        ) : (
          <ResponsiveContainer width="100%" height={350}>
            <DynamicBarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip content={<CustomTooltip />} />
              <Legend />
              <DynamicBar dataKey="value" name="Количество записей" radius={[8, 8, 0, 0]}>
                {chartData.map((entry, index) => (
                  <DynamicCell
                    key={`cell-${index}`}
                    fill={CONFIDENCE_COLORS[entry.name as keyof typeof CONFIDENCE_COLORS]}
                  />
                ))}
              </DynamicBar>
            </DynamicBarChart>
          </ResponsiveContainer>
        )}

        {/* Детальная статистика */}
        <div className="mt-6 space-y-3">
          <div className="text-sm font-medium text-muted-foreground mb-2">Детальная разбивка:</div>
          {validData.length > 0 ? validData.map((item) => (
            <div key={item.range} className="flex items-center justify-between p-3 border rounded-lg">
              <div className="flex items-center gap-3">
                <div
                  className="w-4 h-4 rounded"
                  style={{
                    backgroundColor: CONFIDENCE_COLORS[item.range as keyof typeof CONFIDENCE_COLORS]
                  }}
                />
                <div>
                  <div className="font-medium">{item.range}</div>
                  <div className="text-sm text-muted-foreground">
                    {isNaN(item.count) || item.count === undefined ? 'не число записей' : `${item.count.toLocaleString()} записей`}
                  </div>
                </div>
              </div>
              <Badge variant={item.percentage > 20 ? "default" : "secondary"}>
                {isNaN(item.percentage) || item.percentage === undefined ? 'NaN%' : `${item.percentage.toFixed(1)}%`}
              </Badge>
            </div>
          )) : (
            <div className="text-sm text-muted-foreground text-center py-4">
              Нет данных для отображения
            </div>
          )}
        </div>

        {/* Рекомендации */}
        <div className="mt-6 pt-4 border-t">
          <div className="text-sm font-medium mb-2">Анализ:</div>
          <div className="space-y-2 text-sm text-muted-foreground">
            {(() => {
              const highConfidence = validData.find(d => d.range === '0.9-1.0')?.percentage || 0;
              const mediumConfidence = validData.find(d => d.range === '0.7-0.9')?.percentage || 0;
              const lowConfidence = validData.filter(d => parseFloat(d.range.split('-')[0]) < 0.7)
                .reduce((sum, d) => sum + d.percentage, 0);

              return (
                <>
                  {highConfidence > 50 && (
                    <p className="text-green-600">
                      ✓ Отлично! {highConfidence.toFixed(1)}% записей имеют высокую уверенность (&gt;0.9)
                    </p>
                  )}
                  {mediumConfidence > 30 && (
                    <p className="text-blue-600">
                      • {mediumConfidence.toFixed(1)}% записей имеют среднюю уверенность (0.7-0.9)
                    </p>
                  )}
                  {lowConfidence > 20 && (
                    <p className="text-red-600">
                      ⚠ Внимание: {lowConfidence.toFixed(1)}% записей имеют низкую уверенность (&lt;0.7).
                      Рекомендуется проверить правила классификации.
                    </p>
                  )}
                </>
              );
            })()}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
