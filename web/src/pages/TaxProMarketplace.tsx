import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { BadgeCheck, Star, Clock, FileText, Search } from 'lucide-react'

const EXPERTS = [
    {
        id: 'e1',
        name: 'Oluwaseun Adebayo, FCA',
        title: 'Senior Chartered Accountant (ICAN)',
        rating: 4.9,
        reviews: 142,
        img: 'https://i.pravatar.cc/150?u=a042581f4e29026704d',
        expertise: ['Audit Defense', 'Expat Taxes', 'Corporate'],
        availability: 'Available today',
        price: '₦25,000 / review'
    },
    {
        id: 'e2',
        name: 'Chioma Nnamdi, ACTI',
        title: 'Certified Tax Practitioner (CITN)',
        rating: 5.0,
        reviews: 89,
        img: 'https://i.pravatar.cc/150?u=a042581f4e29026024d',
        expertise: ['Crypto Assets', 'Freelancers', 'SMEs'],
        availability: 'Available tomorrow',
        price: '₦15,000 / review'
    },
    {
        id: 'e3',
        name: 'Ibrahim Hassan, MNIM',
        title: 'Ex-FIRS Assessment Officer',
        rating: 4.8,
        reviews: 210,
        img: 'https://i.pravatar.cc/150?u=a048581f4e29026701d',
        expertise: ['High Net Worth', 'Real Estate', 'Disputes'],
        availability: 'Usually responds in 1 hour',
        price: '₦35,000 / review'
    },
    {
        id: 'e4',
        name: 'Grace Okon, FCA',
        title: 'Tax Strategist & Chartered Accountant',
        rating: 4.9,
        reviews: 56,
        img: 'https://i.pravatar.cc/150?u=a04258a2462d826712d',
        expertise: ['Startups', 'Tech Workers', 'Foreign Income'],
        availability: 'Available today',
        price: '₦20,000 / review'
    }
]

export function TaxProMarketplace() {
    return (
        <DashboardLayout>
            <div className="space-y-10 pb-24 animate-in fade-in slide-in-from-bottom-4 duration-500">

                {/* Header */}
                <div className="text-center max-w-2xl mx-auto pt-8">
                    <BadgeCheck className="w-12 h-12 text-blue-600 mx-auto mb-4" />
                    <h1 className="text-4xl font-black text-neutral-900 tracking-tight leading-tight">
                        Hand off your return to a verified Nigerian Tax Pro.
                    </h1>
                    <p className="text-neutral-500 mt-4 text-lg leading-relaxed">
                        Every expert on our platform is licensed by CITN or ICAN. They'll review every line, find missing deductions, and sign off on your return directly.
                    </p>

                    <div className="mt-8 flex items-center bg-white p-2 rounded-2xl shadow-sm border border-neutral-200 focus-within:border-blue-500 transition-colors mx-auto max-w-md">
                        <Search className="w-5 h-5 text-neutral-400 ml-3" />
                        <input
                            type="text"
                            placeholder="Search by expertise (e.g. Crypto, Freelance, Appeals)"
                            className="w-full bg-transparent border-none focus:outline-none p-3 text-neutral-900 placeholder:text-neutral-400"
                        />
                        <button className="h-11 px-5 bg-neutral-900 text-white font-bold rounded-xl hover:bg-neutral-800 transition-colors">
                            Find Match
                        </button>
                    </div>
                </div>

                {/* Feature Strip */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6 pt-6 border-t border-neutral-100">
                    <div className="text-center">
                        <h3 className="font-bold text-neutral-900 mb-1">100% Accurate Guarantee</h3>
                        <p className="text-sm text-neutral-500">We pay any NRS penalties if there's an error.</p>
                    </div>
                    <div className="text-center md:border-l md:border-r border-neutral-100">
                        <h3 className="font-bold text-neutral-900 mb-1">Secure Document Vault</h3>
                        <p className="text-sm text-neutral-500">Your Form H1 and slips securely shared.</p>
                    </div>
                    <div className="text-center">
                        <h3 className="font-bold text-neutral-900 mb-1">Direct e-Filing</h3>
                        <p className="text-sm text-neutral-500">The Pro files your return straight to the NRS.</p>
                    </div>
                </div>

                {/* Expert Grid */}
                <h2 className="text-2xl font-bold text-neutral-900 pt-4">Top Rated Pros this week</h2>
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {EXPERTS.map(expert => (
                        <div key={expert.id} className="bg-white border border-neutral-200 rounded-3xl p-6 shadow-sm hover:shadow-lg transition-shadow flex flex-col md:flex-row gap-6">

                            <div className="shrink-0 flex flex-col items-center">
                                <div className="relative">
                                    <img src={expert.img} alt={expert.name} className="w-24 h-24 rounded-full object-cover border-4 border-white shadow-md relative z-10" />
                                    <div className="absolute bottom-0 right-0 z-20 bg-blue-600 text-white p-1 rounded-full border-2 border-white shadow-sm">
                                        <BadgeCheck className="w-4 h-4" />
                                    </div>
                                </div>
                                <div className="flex items-center gap-1 mt-3">
                                    <Star className="w-4 h-4 text-amber-400 fill-amber-400" />
                                    <span className="font-bold text-neutral-900 text-sm">{expert.rating}</span>
                                    <span className="text-xs text-neutral-500">({expert.reviews})</span>
                                </div>
                            </div>

                            <div className="flex-1 min-w-0">
                                <h3 className="text-[19px] font-bold text-neutral-900 truncate">{expert.name}</h3>
                                <p className="text-sm text-blue-600 font-semibold mb-3">{expert.title}</p>

                                <div className="flex flex-wrap gap-2 mb-4">
                                    {expert.expertise.map(exp => (
                                        <span key={exp} className="px-2 py-1 bg-neutral-100 text-neutral-600 text-[11px] font-bold uppercase tracking-wider rounded-md">
                                            {exp}
                                        </span>
                                    ))}
                                </div>

                                <div className="flex items-center gap-4 text-xs text-neutral-500 mb-6">
                                    <div className="flex items-center gap-1.5 font-medium">
                                        <Clock className="w-4 h-4 text-emerald-500" /> {expert.availability}
                                    </div>
                                    <div className="flex items-center gap-1.5 font-medium">
                                        <FileText className="w-4 h-4 text-neutral-400" /> {expert.price}
                                    </div>
                                </div>

                                <div className="flex items-center gap-3">
                                    <button className="flex-1 h-11 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-xl transition-colors">
                                        Request Review
                                    </button>
                                    <button className="px-4 h-11 bg-neutral-100 hover:bg-neutral-200 text-neutral-700 font-semibold rounded-xl transition-colors shrink-0">
                                        View Profile
                                    </button>
                                </div>
                            </div>

                        </div>
                    ))}
                </div>
            </div>
        </DashboardLayout>
    )
}
