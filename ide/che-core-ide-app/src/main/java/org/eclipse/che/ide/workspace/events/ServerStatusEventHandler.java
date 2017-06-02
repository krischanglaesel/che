/*******************************************************************************
 * Copyright (c) 2012-2017 Codenvy, S.A.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *   Codenvy, S.A. - initial API and implementation
 *******************************************************************************/
package org.eclipse.che.ide.workspace.events;

import com.google.inject.Inject;
import com.google.inject.Singleton;
import com.google.web.bindery.event.shared.EventBus;

import org.eclipse.che.api.core.jsonrpc.commons.RequestHandlerConfigurator;
import org.eclipse.che.api.workspace.shared.dto.event.ServerStatusEvent;
import org.eclipse.che.ide.api.app.AppContext;
import org.eclipse.che.ide.api.machine.ServerRunningEvent;
import org.eclipse.che.ide.api.machine.ServerStoppedEvent;
import org.eclipse.che.ide.api.machine.WsAgentServerRunningEvent;
import org.eclipse.che.ide.api.machine.WsAgentServerStoppedEvent;
import org.eclipse.che.ide.api.machine.events.WsAgentStateEvent;
import org.eclipse.che.ide.api.machine.events.WsAgentStateHandler;
import org.eclipse.che.ide.api.workspace.model.RuntimeImpl;
import org.eclipse.che.ide.api.workspace.model.WorkspaceImpl;
import org.eclipse.che.ide.context.AppContextImpl;
import org.eclipse.che.ide.util.loging.Log;
import org.eclipse.che.ide.workspace.WorkspaceServiceClient;

import static org.eclipse.che.api.core.model.workspace.runtime.ServerStatus.RUNNING;
import static org.eclipse.che.api.core.model.workspace.runtime.ServerStatus.STOPPED;
import static org.eclipse.che.api.machine.shared.Constants.WSAGENT_REFERENCE;

@Singleton
class ServerStatusEventHandler {

    @Inject
    ServerStatusEventHandler(RequestHandlerConfigurator configurator,
                             EventBus eventBus,
                             AppContext appContext,
                             WorkspaceServiceClient workspaceServiceClient) {
        configurator.newConfiguration()
                    .methodName("server/statusChanged")
                    .paramsAsDto(ServerStatusEvent.class)
                    .noResult()
                    .withBiConsumer((endpointId, event) -> {
                        Log.debug(getClass(), "Received notification from endpoint: " + endpointId);

                        workspaceServiceClient.getWorkspace(appContext.getWorkspaceId()).then(workspace -> {
                            ((AppContextImpl)appContext).setWorkspace(workspace);

                            if (event.getStatus() == RUNNING) {
                                eventBus.fireEvent(new ServerRunningEvent(event.getServerName(), event.getMachineName()));

                                if (WSAGENT_REFERENCE.equals(event.getServerName())) {
                                    eventBus.fireEvent(new WsAgentServerRunningEvent());
                                }
                            } else if (event.getStatus() == STOPPED) {
                                eventBus.fireEvent(new ServerStoppedEvent(event.getServerName(), event.getMachineName()));

                                if (WSAGENT_REFERENCE.equals(event.getServerName())) {
                                    eventBus.fireEvent(new WsAgentServerStoppedEvent());
                                }
                            }
                        });
                    });
    }

    // TODO: remove when real events are finished
    @Inject
    private void fakeRunningAllServers(EventBus eventBus, AppContext appContext) {
        eventBus.addHandler(WsAgentStateEvent.TYPE, new WsAgentStateHandler() {
            @Override
            public void onWsAgentStarted(WsAgentStateEvent event) {
                final WorkspaceImpl workspace = appContext.getWorkspace();
                final RuntimeImpl runtime = workspace.getRuntime();

                if (runtime != null) {
                    runtime.getMachines()
                           .values()
                           .forEach(machine -> machine
                                   .getServers()
                                   .values()
                                   .forEach(server -> {
                                       eventBus.fireEvent(new ServerRunningEvent(server.getName(), machine.getName()));
                                   }));
                }
                eventBus.fireEvent(new WsAgentServerRunningEvent());
            }

            @Override
            public void onWsAgentStopped(WsAgentStateEvent event) {
            }
        });
    }
}
