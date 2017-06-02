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
package org.eclipse.che.ide.machine;

import com.google.inject.Inject;
import com.google.inject.Singleton;
import com.google.web.bindery.event.shared.EventBus;

import org.eclipse.che.ide.api.app.AppContext;
import org.eclipse.che.ide.api.machine.WsAgentURLModifier;
import org.eclipse.che.ide.api.workspace.event.WorkspaceStartedEvent;
import org.eclipse.che.ide.api.workspace.model.MachineImpl;
import org.eclipse.che.ide.api.workspace.model.WorkspaceImpl;

import java.util.Optional;

/** Initializer for {@link WsAgentURLModifier}. */
@Singleton
public class WsAgentURLModifierInitializer {

    private final WsAgentURLModifier wsAgentURLModifier;

    @Inject
    public WsAgentURLModifierInitializer(WsAgentURLModifier wsAgentURLModifier,
                                         AppContext appContext,
                                         EventBus eventBus) {
        this.wsAgentURLModifier = wsAgentURLModifier;

        final WorkspaceImpl workspace = appContext.getWorkspace();
        final Optional<MachineImpl> devMachine = workspace.getDevMachine();

        if (devMachine.isPresent()) {
            init(devMachine.get());
        } else {
            // subscribe to machine status
            eventBus.addHandler(WorkspaceStartedEvent.TYPE, e -> init(devMachine.get()));
        }
    }

    private void init(MachineImpl devMachine) {
        wsAgentURLModifier.initialize(devMachine);
    }
}
