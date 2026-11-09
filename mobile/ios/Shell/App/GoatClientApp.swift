// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 U.S. Federal Government (in countries where recognized).

import SwiftUI

@main
struct GoatClientApp: App {
    @StateObject private var tunnel = TunnelManager()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(tunnel)
                .task {
                    await tunnel.loadFromSystem()
                }
        }
    }
}
