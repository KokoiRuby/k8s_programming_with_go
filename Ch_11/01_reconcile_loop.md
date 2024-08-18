The **Reconcile** func contains all the **business logic** of the Operator. 

The role of Reconcile func is to ensure that the **state** of the system **matches** what is specified in the resource to be reconciled.

The **Reconcile Loop** is known as the process of **watching for resources and calling the Reconcile function** when resources are created, modified, or deleted, **implementing the reconciled resource with low-level ones**, watching for the status of these resources, and updating the status of the reconciled resource accordingly.

Nore: “low-level” holds **OwnerReferences** of “high-level”.

## Writing the Reconcile Function

The Reconcile func receives from the **queue** a resource to reconcile.

The **first** operation to work on is to get the information about this resource to reconcile.

Only **obj-key = namespece/obj-name** will be dequeued. (as designed, see more in Informer)

### Checking Whether the Resource Exists

Enqueued due to created/modified, Get will work; Notfound err if deleted.

The good practice for an Operator is to **add OwnerReferences** to resources it creates.

Once primary resource is deleted, low-level resources will be deleted by GC automatically.

### Implementing the Reconciled Resource

Once resource to reconcile is found, the **next step** is to create "low-level" to impl.

A good way is is to **declare what the low-level resources should be** & let kube-controller to take over & reconcile them.

One could check whether resources exist, and create them if they do not, or modify them if they exist.

**Server-side Apply** is a perfect choice: patch if exists, create if not, conflict-free.

After Apply

1. If the low-level resources already exist and have not been modified by the Apply, no **MODIFIED event** will be triggered for these resources, and the **Reconcile func** will not be called again.
2. If the low-level resources have been created or modified, this will trigger **CREATED or MODIFIED events** for these resources, and the Reconcile function will be called again because of these events.

The Operator needs, at some point, to read the Statuses of the low-level it has created so as to compute the status of the reconciled resource.

## Simple Example

To creates a **Deployment** with the Image and Memory information provided in a **MyResource** instance.