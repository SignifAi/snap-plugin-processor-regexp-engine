Name:           signifai-snap-plugin-processor-regexp-engine
Version:        VERSION
Release:        1%{?dist}
Summary:        Snap Telemetry Agent

License:        Apache
URL:            https://github.com/signifai/snap-plugin-processor-regexp-engine
# Fetch this tarball this way:
#   curl -L https://github.com/SignifAi/snap-plugin-processor-regexp-engine/archive/v1.0.0.tar.gz -o signifai-snap-plugin-processor-regexp-engine-1.0.0.tar.gz
Source0:        snap-plugin-processor-regexp-engine

Requires:       signifai-go >= 1.8.3-el6.1
Requires:       signifai-snap-agent >= 1.2.0-el6.1

%description


%prep
# No prep; already done

%build
# No build; we already did that.

%install
rm -rf $RPM_BUILD_ROOT

mkdir -p $RPM_BUILD_ROOT/opt/signifai/snap/plugins
cp %{SOURCE0} $RPM_BUILD_ROOT/opt/signifai/snap/plugins/snap-plugin-processor-regexp-engine
%clean


%files
%defattr(-,root,root,-)
/opt/signifai/snap/plugins/snap-plugin-processor-regexp-engine

%changelog
