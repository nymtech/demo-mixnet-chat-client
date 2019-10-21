const path = require('path')
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const HtmlWebpackTagsPlugin = require('html-webpack-tags-plugin');

const commonConfig = {
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: '[name].js'
  },
  module: {
    rules: [
      {
        test: /\.js$/,
        enforce: 'pre',
        loader: 'standard-loader',
        options: {
          typeCheck: true,
          emitErrors: true
        }
      },
      {
        test: /\.tsx?$/,
        loader: 'babel-loader'
      },
      {
        test:/\.css$/,
        use: [MiniCssExtractPlugin.loader, 'css-loader'],
      },
    ]
  },
}

const HtmlWebpackPlugin = require('html-webpack-plugin')
module.exports = [
  Object.assign(
    {
      target: 'electron-main',
      entry: { main: './src/main.ts' }
    },
    commonConfig),
  Object.assign(
    {
      target: 'electron-renderer',
      entry: { renderer: './src/renderer.ts' },
      plugins: [
        new HtmlWebpackPlugin({
          template: 'src/index.html',
        }),
        new HtmlWebpackTagsPlugin({ tags: ['styles.css'], append: true }),
        new MiniCssExtractPlugin({
          chunkFilename: '[name].css',
          filename: 'styles.css'
      })
      ]
    },
    commonConfig),
]