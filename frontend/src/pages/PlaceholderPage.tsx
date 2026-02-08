import { useTheme } from '../contexts/ThemeContext'

const PlaceholderPage = ({ title }: { title: string }) => {
  const { theme } = useTheme()
  
  return (
    <div className="min-h-screen bg-base-200 p-8">
      <div className="max-w-4xl mx-auto">
        <div className="card bg-base-100 shadow-lg">
          <div className="card-body">
            <h1 className="text-3xl font-bold mb-4">{title}</h1>
            <p className="text-base-content/70">
              此页面正在从 Ant Design 迁移到 daisyUI，敬请期待。
            </p>
            <div className="mt-6">
              <div className="badge badge-info">当前主题: {theme}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default PlaceholderPage