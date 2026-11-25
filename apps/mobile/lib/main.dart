import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/tokens/app_colors.dart';

// ETHER Phase 01 骨架：最小可啟動殼。
// 只有一個佔位路由、一個佔位頁；無功能、無按鈕、無假資料。
// ThemeData 只引用已定的 8 個品牌色（見 docs/brand/DESIGN_TOKENS.md §1.1），
// 其餘 token 待 Phase 02 定稿，不得編造。
// TODO(Phase 02): Design Tokens 定稿後，以 design_system 元件庫取代此處的基底設定。

final GoRouter _router = GoRouter(
  routes: <RouteBase>[
    GoRoute(
      path: '/',
      builder: (BuildContext context, GoRouterState state) {
        return const Scaffold(
          body: Center(
            child: Text('Phase 01 骨架，尚無功能'),
          ),
        );
      },
    ),
  ],
);

void main() {
  runApp(const ProviderScope(child: EtherApp()));
}

class EtherApp extends StatelessWidget {
  const EtherApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'ETHER',
      theme: ThemeData(
        useMaterial3: true,
        scaffoldBackgroundColor: AppColors.pureLight,
        colorScheme: const ColorScheme.light(
          primary: AppColors.etherealBlue,
          secondary: AppColors.auroraCyan,
          tertiary: AppColors.softViolet,
          surface: AppColors.pureLight,
          onPrimary: AppColors.etherMidnight,
          onSecondary: AppColors.etherMidnight,
          onSurface: AppColors.etherMidnight,
        ),
        textTheme: const TextTheme(
          bodyMedium: TextStyle(color: AppColors.etherMidnight),
        ),
      ),
      darkTheme: ThemeData(
        useMaterial3: true,
        scaffoldBackgroundColor: AppColors.etherMidnight,
        colorScheme: const ColorScheme.dark(
          primary: AppColors.etherealBlue,
          secondary: AppColors.auroraCyan,
          tertiary: AppColors.softViolet,
          surface: AppColors.deepNebula,
          onPrimary: AppColors.pureLight,
          onSecondary: AppColors.pureLight,
          onSurface: AppColors.pureLight,
        ),
        textTheme: const TextTheme(
          bodyMedium: TextStyle(color: AppColors.pureLight),
        ),
      ),
      routerConfig: _router,
    );
  }
}
