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
package org.eclipse.che.ide.api.machine;

import com.google.gwt.event.shared.EventHandler;
import com.google.gwt.event.shared.GwtEvent;

/** Fired when a server in a machine is run. */
public class ServerRunningEvent extends GwtEvent<ServerRunningEvent.Handler> {

    public static final Type<ServerRunningEvent.Handler> TYPE = new Type<>();

    private final String serverName;
    private final String machineName;

    public ServerRunningEvent(String serverName, String machineName) {
        this.serverName = serverName;
        this.machineName = machineName;
    }

    public String getServerName() {
        return serverName;
    }

    public String getMachineName() {
        return machineName;
    }

    @Override
    public Type<Handler> getAssociatedType() {
        return TYPE;
    }

    @Override
    protected void dispatch(Handler handler) {
        handler.onServerRunning(this);
    }

    public interface Handler extends EventHandler {
        void onServerRunning(ServerRunningEvent event);
    }
}
